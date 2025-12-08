# Local RAG for markdown documentation: a complete guide for CLI-first tooling

Building a CLI-first RAG tool for local markdown documentation is now highly feasible with mature, self-hosted components. The best approach combines **sqlite-vec or LanceDB** for vector storage, **BGE or Nomic embedding models**, **header-based markdown chunking**, and an **MCP server interface** for AI agent integration. Google's Gemini context caching is cloud-only and unsuitable for self-hosted use, but local long-context models like Qwen2.5-1M offer alternatives for smaller documentation sets.

## Gemini's context caching cannot be self-hosted, but alternatives exist

Google's context caching feature, released in May 2024 and enhanced with implicit caching in May 2025, stores frequently-used input tokens server-side across API requests. It offers **90% cost reduction** on cached tokens for Gemini 2.5 models and supports 1-2 million token context windows. However, it requires Google's cloud infrastructure—caches exist on Google servers, files must be uploaded via their Files API, and there is no offline capability.

For a self-hosted approach, two paths are available:

**Long-context local models** can fit hundreds of markdown files directly in context without embedding-based retrieval:
- **Qwen2.5-1M**: First open-source 1M token context model, achieves 93.1 on RULER benchmark (vs GPT-4's 91.6)
- **Llama 3.1 (8B/70B/405B)**: 128K context with Apache-style license
- **Gemma 3 27B**: Google's open-weight model with 128K context

These run locally via Ollama (`ollama run qwen2.5:72b`) or vLLM. For documentation sets under **~750K words** (≈1M tokens), stuffing everything into context may outperform RAG for coherence and simplicity. However, RAG remains essential when you exceed context limits, need citation accuracy, or want sub-second query latency.

## sqlite-vec and BGE models form the best local embedding stack

For hundreds to low thousands of markdown files, **sqlite-vec** combined with **BGE-small-en-v1.5** or **nomic-embed-text-v1.5** offers the optimal balance of portability, performance, and simplicity.

### Local embedding model recommendations

| Model | Dimensions | Best for | Key advantage |
|-------|------------|----------|---------------|
| **BGE-M3** | 768 | Multi-lingual, long docs | Dense + sparse + ColBERT in one model, 8K context |
| **nomic-embed-text-v1.5** | 64-768 (flexible) | Accuracy-critical | Matryoshka embeddings allow dimension tuning, 8192 tokens |
| **BGE-small-en-v1.5** | 384 | Balance of speed/quality | Well-integrated with LlamaIndex/LangChain |
| **all-MiniLM-L6-v2** | 384 | Speed-critical | 5x faster, ONNX-optimized versions available |

For **documentation with code examples**, BGE-M3 handles technical content well without requiring separate code-specific models. Nomic-embed-text requires task prefixes (`search_document:`, `search_query:`) but achieves 86.2% accuracy on retrieval benchmarks.

**ONNX quantization** delivers 2-3x speedups on CPU:
```python
from sentence_transformers import SentenceTransformer
model = SentenceTransformer("BAAI/bge-small-en-v1.5", backend="onnx")
```

### Vector store comparison for local deployment

**sqlite-vec** (6.4K GitHub stars) is the recommended SQLite-native option—pure C with zero dependencies, runs everywhere including WASM, and handles **100K vectors in under 20ms** without indexing. Its predecessor sqlite-vss is deprecated.

```python
import sqlite3, sqlite_vec
conn = sqlite3.connect("docs.db")
sqlite_vec.load(conn)
conn.execute("CREATE VIRTUAL TABLE docs USING vec0(embedding float[384], +text TEXT, +project TEXT, +file_path TEXT)")
```

**LanceDB** offers superior scalability for larger collections—disk-based indexes that scale beyond RAM, hybrid vector + full-text search, and handles **200M+ vectors**. It's YC-backed and production-ready.

**ChromaDB** (17K stars) provides the simplest Python API with built-in embedding functions, but shows **storage overhead** (~10GB vs 1GB for equivalent pgvector datasets) and degraded concurrent performance.

For your use case (hundreds to thousands of files, multi-project), sqlite-vec or LanceDB both work excellently. Choose sqlite-vec for maximum portability and SQLite ecosystem benefits; choose LanceDB if you anticipate significant growth or want built-in hybrid search.

## Header-based chunking is essential for markdown documentation

Treating markdown as flat text loses critical structure. The recommended approach uses **two-stage chunking**: first split by headers to preserve document organization, then apply size constraints within sections.

### Optimal chunking strategy

```python
from langchain_text_splitters import MarkdownHeaderTextSplitter, RecursiveCharacterTextSplitter

# Stage 1: Preserve structure
headers_to_split_on = [("#", "h1"), ("##", "h2"), ("###", "h3")]
md_splitter = MarkdownHeaderTextSplitter(headers_to_split_on=headers_to_split_on, strip_headers=False)
header_splits = md_splitter.split_text(markdown_content)

# Stage 2: Size constraints
text_splitter = RecursiveCharacterTextSplitter(chunk_size=500, chunk_overlap=50)
final_chunks = text_splitter.split_documents(header_splits)
```

Key parameters based on 2024-2025 research:
- **Chunk size**: 400-500 tokens for technical documentation (Chroma research found 400 tokens with text-embedding-3-large achieved 88-89% recall)
- **Overlap**: 10-15% of chunk size (~50-75 tokens)
- **Headers**: Include in content AND metadata for retrieval context
- **Code blocks**: Never split mid-block; use `strip_headers=False` to preserve context

### Code block handling is critical

LangChain's `RecursiveCharacterTextSplitter.from_language()` respects function/class boundaries:
```python
from langchain_text_splitters import Language, RecursiveCharacterTextSplitter
python_splitter = RecursiveCharacterTextSplitter.from_language(
    language=Language.PYTHON, chunk_size=500, chunk_overlap=50
)
```

LlamaIndex's `CodeSplitter` provides similar functionality with explicit control over chunk lines and overlap.

### Metadata to preserve

Every chunk should include:
- **Header hierarchy**: `{"h1": "API Reference", "h2": "Authentication", "h3": "OAuth2"}`
- **File path**: Relative path within project
- **Project ID**: For multi-project filtering
- **Source URL**: If applicable, for citation
- **Front matter fields**: Parse with `python-frontmatter` and include as filterable metadata

## LlamaIndex CLI leads for framework-based solutions, txtai for simplicity

Among RAG frameworks, **LlamaIndex** offers the most CLI-friendly experience with `llamaindex-cli rag`, while **txtai** provides the simplest all-in-one local solution.

### LlamaIndex (recommended for customization)

```bash
# Built-in CLI for quick prototyping
llamaindex-cli rag --files "./docs/**/*.md" --chat

# Or programmatic use with full control
from llama_index.core import SimpleDirectoryReader, VectorStoreIndex
docs = SimpleDirectoryReader("data", recursive=True, required_exts=[".md"]).load_data()
index = VectorStoreIndex.from_documents(docs)
query_engine = index.as_query_engine()
```

LlamaIndex strengths: excellent markdown handling via SimpleDirectoryReader, ColBERT integration via RAGatouille pack, strong Ollama support for local models. Main weakness: no built-in "project" concept—you manage separate indexes manually.

### txtai (recommended for simplicity)

txtai achieves near-zero configuration for local RAG:
```python
from txtai import Embeddings, RAG
embeddings = Embeddings(content=True)
embeddings.index(documents)
rag = RAG(embeddings, "meta-llama/Meta-Llama-3.1-8B-Instruct")
response = rag("What is the main feature?")
```

It includes a built-in API server (`uvicorn "txtai.api:app"`) and Docker deployment. For the simplest path to a working local RAG system, txtai is hard to beat.

### LangChain assessment

LangChain provides the largest ecosystem (150+ data loaders) but adds significant complexity for simple RAG tasks. Multiple packages must be installed (`langchain`, `langchain-community`, `langchain-chroma`, `langchain-text-splitters`), and the abstraction layers add overhead. Use LangChain only if you need its extensive integrations or already have LangChain expertise.

### Framework comparison for your use case

| Framework | CLI support | Project management | Setup complexity | Best for |
|-----------|-------------|-------------------|-----------------|----------|
| **LlamaIndex** | ✅ Built-in CLI | Via named indexes | Medium | Customization, production |
| **txtai** | ✅ API server | Via embeddings indexes | Low | Rapid prototyping |
| **LangChain** | ❌ None | Via stores | High | Complex integrations |
| **Haystack** | ❌ None | Via pipelines | Medium | Enterprise RAG |

## Emerging tools prioritize MCP integration and local-first operation

The most significant development for AI coding agent integration is **MCP (Model Context Protocol)**, released by Anthropic in late 2024. Several RAG tools now implement MCP servers, enabling Claude, Cursor, and other MCP-compatible clients to query documentation directly.

### Notable tools for developer-focused RAG

**Khoj** (self-hostable "AI second brain") supports chat with local/online LLMs, accesses documentation from Obsidian/Notion/files, and includes research mode with `/research` command. It's actively maintained and integrates with WhatsApp, Emacs, and browsers.

**AnythingLLM** (47K+ GitHub stars) provides a polished desktop app with built-in Ollama and LanceDB. Its **workspace concept** maps directly to documentation projects—each workspace is isolated with its own documents, supporting both query and conversation modes. MCP-compatible as of 2025.

**gptme-rag** offers a reference CLI design: `gptme-rag index`, `gptme-rag search`, `gptme-rag watch`. It uses ChromaDB and includes file watching with token-aware context assembly.

### Reranking significantly improves retrieval quality

Adding a reranker after initial vector retrieval improves accuracy by 10-20%. The **rerankers** library from Answer.AI provides a unified API:
```python
from rerankers import Reranker
ranker = Reranker("mixedbread-ai/mxbai-rerank-v2")  # Current open-source SOTA
results = ranker.rank(query, initial_results)
```

Local options include BGE-reranker-v2-m3, Jina-reranker-v3 (supports 131K context), and ColBERT-based late interaction models. For code-heavy documentation, Jina-reranker-v2 includes code retrieval capabilities.

## Multi-project architecture favors single store with metadata filtering

For organizing discrete documentation projects (e.g., docs.tilt.dev vs inngest.com/docs), the recommended pattern is a **single vector store with metadata filtering** rather than separate stores per project.

### Why single-store with metadata

- **Cost-efficient**: No per-project infrastructure overhead
- **Cross-project queries**: "Search all my docs for X" works naturally
- **Simpler management**: One database to back up, migrate, and maintain
- **Consistent chunking**: Most markdown docs share similar structure

Separate stores make sense only when projects require fundamentally different chunking strategies or strict isolation for compliance reasons.

### Database schema pattern

```sql
-- Projects table
CREATE TABLE projects (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  source_path TEXT NOT NULL,
  config JSON -- chunking settings, patterns
);

-- Documents with file hash for incremental indexing
CREATE TABLE documents (
  id TEXT PRIMARY KEY,
  project_id TEXT REFERENCES projects(id),
  file_path TEXT NOT NULL,
  file_hash TEXT NOT NULL,
  UNIQUE(project_id, file_path)
);

-- Chunks with denormalized project_id for filter performance
CREATE VIRTUAL TABLE chunks USING vec0(
  embedding float[384],
  +document_id TEXT,
  +project_id TEXT,
  +content TEXT,
  +metadata JSON
);
```

Query with project filtering:
```python
results = vector_store.query(
    query_embedding=embed(query),
    filter={"project_id": "tilt-docs"},
    top_k=10
)
```

### Recommended CLI structure

```bash
docrag init                          # Initialize configuration
docrag add-project tilt-docs /path/to/docs  # Add documentation source
docrag index [--project tilt-docs]   # Index all or specific project
docrag search "query" --project tilt-docs --format json
docrag watch                         # File watching with debounced reindex
docrag serve-mcp                     # Start MCP server for Claude Code
```

Output formats should include `--format json` for AI agent consumption and `--format context` for ready-to-use LLM context blocks.

### MCP server for Claude Code integration

An MCP server exposes your RAG tool as a callable resource:

```python
from mcp.server import Server
from mcp.types import Tool, TextContent

server = Server("docrag-mcp")

@server.list_tools()
async def list_tools():
    return [Tool(
        name="search_docs",
        description="Search local documentation",
        inputSchema={
            "type": "object",
            "properties": {
                "query": {"type": "string"},
                "project": {"type": "string"},
                "top_k": {"type": "integer", "default": 5}
            },
            "required": ["query"]
        }
    )]
```

Configure in `.mcp.json` for Claude Code:
```json
{
  "mcpServers": {
    "docrag": {
      "command": "python",
      "args": ["/path/to/docrag/mcp_server.py"]
    }
  }
}
```

## Long-context versus embedding-based RAG tradeoffs

| Factor | Long-context (stuffing docs) | Embedding-based RAG |
|--------|------------------------------|---------------------|
| **Setup complexity** | Low (no embedding pipeline) | Medium (embeddings, vector store) |
| **Cost** | Higher per query (more tokens) | Lower per query, upfront indexing cost |
| **Latency** | Higher (process all tokens) | Lower (retrieve then generate) |
| **Coherence** | Better (full context available) | May miss cross-document relationships |
| **Citation accuracy** | Lower ("lost in the middle" problem) | Higher (explicit source tracking) |
| **Scale limit** | ~1-2M tokens (~750K-1.5M words) | Essentially unlimited |
| **Incremental updates** | Easy (just load new docs) | Requires reindexing changed files |

**Recommendation**: For documentation sets under 500K words with stable content, consider long-context approaches with local models like Qwen2.5-1M. For larger, frequently-updated documentation, or when citation accuracy matters, use embedding-based RAG with reranking.

A **hybrid approach** works well: use RAG to retrieve the top 20-50 relevant chunks, then pass those to a long-context model for synthesis with full awareness of retrieved context.

## Concrete architecture recommendation

For a CLI-first documentation RAG tool targeting AI coding agents:

```
┌─────────────────────────────────────────────────────────────┐
│  CLI Interface: docrag add|index|search|watch|serve-mcp    │
└─────────────────────────────┬───────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────┐
│  Core: ProjectManager | MarkdownIndexer | Searcher | Watcher│
└─────────────────────────────┬───────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────┐
│  Storage: sqlite-vec (vectors) + SQLite (metadata/projects) │
└─────────────────────────────┬───────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────┐
│  Embeddings: BGE-small-en-v1.5 via sentence-transformers    │
│  (ONNX-optimized for CPU inference)                         │
└─────────────────────────────────────────────────────────────┘
```

**Key components**:
1. **sqlite-vec** for vectors (zero dependencies, excellent portability)
2. **BGE-small-en-v1.5** for embeddings (384 dimensions, good quality/speed balance)
3. **Header-based chunking** with 400-500 token chunks, 10% overlap
4. **File hash tracking** for incremental reindexing
5. **MCP server mode** for Claude Code / Cursor integration
6. **JSON output format** for agent consumption

This architecture handles thousands of markdown files efficiently on consumer hardware, provides sub-100ms query latency, and integrates cleanly with AI coding assistants via MCP or direct CLI invocation.
