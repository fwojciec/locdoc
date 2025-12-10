# Ask Command Design

## Overview

The `ask` command allows users to query a project's documentation using natural language. It fetches all documents for the specified project, sends them along with the question to Gemini Flash, and returns the answer as plain text.

## Command Syntax

```
locdoc ask <project> "question"
```

### Example

```
$ locdoc ask inngest "how do I retry a failed step?"

To retry a failed step in Inngest, use the step.retry() option...
```

## Key Decisions

- **No RAG** — Entire documentation corpus sent to LLM. Simpler, more coherent answers. RAG can be added later if needed for very large doc sets.
- **Position-ordered documents** — Documents fetched in sitemap position order (use `SortBy: "position"`). This provides coherent LLM context, with introductory material before advanced topics.
- **Plain text output only** — No JSON or structured output. Users (human or LLM) just want the answer.
- **Gemini Flash** — Chosen for 1M+ token context, explicit caching API (for future use), and low cost.
- **No context caching initially** — Keep it simple. Caching can be layered on if query costs become a concern.
- **Environment variable for API key** — `GEMINI_API_KEY`. Standard pattern, no secrets in config files.

## Architecture

Following Ben Johnson's Standard Package Layout:

### Root Package

New interface in root package:

```go
// Asker provides natural language question answering over documentation.
type Asker interface {
    Ask(ctx context.Context, projectID string, question string) (string, error)
}
```

### gemini/ Package

Implements `Asker` interface. Named after the dependency (Gemini), not the concept.

Responsibilities:
- Wrap the Gemini API client (`google.golang.org/genai`)
- Construct prompts from documents and questions
- Return plain text answers

### cmd/locdoc/

New `CmdAsk` function that:
1. Parses args (project name, question)
2. Looks up project by name via `ProjectService`
3. Fetches all documents for project via `DocumentService`
4. Validates project has documents
5. Calls `Asker.Ask()`
6. Prints result to stdout

## Prompt Construction

```
You are a helpful assistant answering questions about software library documentation.

<documentation>
## Document: {title or source URL}
{content}

## Document: {title or source URL}
{content}
...
</documentation>

Question: {user's question}

Answer based only on the documentation provided. If the answer is not in the documentation, say so.
```

- Documents separated by headers for clarity
- Instruction to stay grounded in provided docs (reduces hallucination)
- Title preferred over URL for readability, fall back to URL if no title

## Error Handling

Actionable error messages for common failure modes:

**Project not found:**
```
project "foo" not found. Use "locdoc list" to see available projects, or "locdoc add <name> <url>" to add one.
```

**Project has no documents:**
```
project "foo" has no documents. Run "locdoc crawl foo" to fetch documentation first.
```

**Missing API key:**
```
GEMINI_API_KEY environment variable not set. Get an API key at https://aistudio.google.com/apikey
```

All errors return exit code 1 and print to stderr.

## Model Selection

Start with `gemini-2.0-flash` (or latest stable Flash model). Hardcoded for now, could become configurable later.

## Future Enhancements (Not in Scope)

- Context caching for reduced costs on repeated queries
- Local model support (Ollama)
- Alternative cloud providers (Claude, OpenAI)
- JSON output format
- Cross-project queries
