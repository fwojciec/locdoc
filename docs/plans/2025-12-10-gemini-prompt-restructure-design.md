# Gemini Prompt Restructure Design

Improve accuracy and enable caching for documentation Q&A by restructuring the Gemini prompt.

## Goals

- **Primary:** Improve answer accuracy, reduce hallucinations
- **Secondary:** Enable implicit caching for cost reduction
- **Deferred:** Verifiability/citations (not a priority now)

## Current State

`gemini/asker.go` uses a simple prompt:
- Inline system instruction
- Single `<documentation>` wrapper with markdown headers
- Question at the end
- Basic constraint about answering only from docs

## Design

### System Instruction (via API field)

```
Answer questions using only the provided documentation.

Rules:
1. Only use information from the provided documents
2. If the answer isn't in the documents, say "I couldn't find this in the documentation"
3. Before answering, internally identify the most relevant passages
4. Be concise and direct
5. Always end with a Sources section listing URLs of documents you referenced
```

Use `genai.GenerateContentConfig.SystemInstruction` instead of inline text.

### User Message Structure

```xml
<documents>
<document index="1">
<title>Page Title</title>
<source>https://example.com/docs/page</source>
<content>
{markdown content}
</content>
</document>
...
</documents>

<question>
{user's question}
</question>

<instructions>
Answer the question based on the documentation above. First internally identify relevant passages, then provide a clear answer.

Format your response as:
[Your answer here]

---
Sources:
- [List URLs of documents you referenced]
</instructions>
```

### Generation Config

```go
&genai.GenerateContentConfig{
    SystemInstruction: &genai.Content{
        Parts: []*genai.Part{{Text: systemInstruction()}},
    },
    Temperature: ptr(0.4),
}
```

### Code Structure

Split current `buildPrompt` into:
1. `systemInstruction()` - returns static system instruction string
2. `formatDocumentsXML(docs)` - serializes documents to XML format
3. `buildUserMessage(docsXML, question)` - wraps with question and trailing instructions

## Rationale

### XML Structure with Metadata
- Research shows XML tags help models "distinguish between instructions, context, and tasks"
- Index enables internal document referencing
- Source URLs allow consuming AI agents to fetch more details if needed
- ~15% token overhead, justified by accuracy improvement

### System Instruction via API
- Google-recommended pattern
- Enables implicit caching (system instruction + documents cache, only question varies)
- 75% cost reduction on cache hits

### Temperature 0.4
- Default 1.0 is inappropriate for factual Q&A
- Research recommends 0.3-0.5 for documentation tasks
- Single most effective lever for reducing hallucinations

### Sandwich Pattern
- Instructions at beginning (system instruction) and end (trailing `<instructions>`)
- Counters primacy/recency effects in long contexts
- Key constraints repeated where they're most likely to be followed

### Internal Quote Extraction
- "Internally identify relevant passages" instruction
- Converts long-context retrieval to focused extraction + reasoning
- Accuracy benefit without output token cost

### Structured Sources Output
- Predictable format for programmatic parsing
- Full URLs so consuming tools can fetch more detail
- Always present (not conditional)

## Document Ordering

Keep current sitemap-based ordering. Relevance-based ordering deferred until embeddings are available.

## Token Impact

- XML structure: +15% vs current markdown
- Source URLs: variable by URL length
- Trailing instructions: ~50 tokens

Offset by implicit caching: 75% discount on repeated queries to same documentation.

## Files to Modify

- `gemini/asker.go` - main implementation
- `gemini/asker_test.go` - update tests

## Out of Scope

- Document indices exposed in output (verifiability deferred)
- Structured JSON output
- Explicit caching API (implicit caching sufficient)
- Relevance-based document ordering (needs embeddings)
- Evals framework (future enhancement)
