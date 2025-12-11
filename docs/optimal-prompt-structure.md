# Optimal prompt structuring for Gemini 2.5 Flash documentation Q&A

Gemini 2.5 Flash's **1M-token context window** makes "stuff everything" approaches viable, but strategic structuring dramatically improves accuracy. The evidence points to a clear optimal pattern: **documents first with XML wrappers, query last, instructions sandwiched at both ends**. Google's own testing shows queries placed at the end improve response quality by up to **30%** for complex multi-document inputs, while the "lost in the middle" problem—though reduced in Gemini 2.5—still causes measurable degradation when critical information sits in the middle of very long contexts. For documentation Q&A specifically, lowering temperature to **0.3–0.5** (not the default 1.0), using explicit grounding instructions, and leveraging context caching can reduce both hallucinations and costs by up to **90%**.

## Document organization directly impacts retrieval accuracy

The ordering of documents within your prompt is not arbitrary—position effects are well-documented and apply even to modern long-context models. Liu et al.'s foundational research ("Lost in the Middle," 2024) established that LLMs exhibit a **U-shaped attention curve**, performing best when relevant information appears at the beginning or end of context, with significant degradation for middle-positioned content. Performance can drop by **20% or more** when critical information moves from privileged positions to the middle.

For Gemini 2.5 Flash specifically, Google reports up to **99% accuracy** on single-needle retrieval tasks, but performance varies for multiple pieces of information. The practical implication: organize documents strategically rather than alphabetically or randomly.

**Evidence-based ordering strategies:**

- **Relevance-based ordering** outperforms other approaches. Research shows a "reverse" configuration—placing more relevant documents closer to the query position at the end—achieved the best RAG scores in comparative testing. If you can estimate relevance (via embedding similarity or URL hierarchy), order from least to most relevant, with most relevant immediately preceding your question.
- **Critical content at both ends** provides insurance against position bias. The "sides repacking" method places the most important documents at both the beginning and end, ensuring at least one instance hits a privileged attention position.
- **Group by document type** when you cannot estimate relevance. Keeping API reference pages together, followed by tutorials, followed by examples, helps the model build coherent mental models of each category. This also improves cache hit rates if users typically ask questions about similar document types.

The "lost in the middle" effect persists in Gemini 2.5 Flash but is less severe than in older models. Gemini 2.5 achieved **94.5% accuracy** on the MRCR benchmark at 128k context length, significantly outperforming competitors on long-context retrieval tasks. Google specifically targeted long-context performance during development, and their internal testing shows "no degradation at extreme lengths" compared to the falloff seen in GPT-4 and Claude.

## XML tags with metadata provide optimal structure

Google's official documentation explicitly recommends XML-style tags for complex prompts, stating they help the model "distinguish between instructions, context, and tasks." Anthropic's research confirms XML as the only format that all major providers—Google, Anthropic, and OpenAI—actively encourage. For your documentation Q&A system, wrap each document individually rather than using a single `<documentation>` wrapper.

**Recommended document structure:**

```xml
<documents>
  <document index="1">
    <source>/api/authentication</source>
    <title>Authentication API Reference</title>
    <content>
      [Full markdown content of page]
    </content>
  </document>
  <document index="2">
    <source>/guides/getting-started</source>
    <title>Getting Started Guide</title>
    <content>
      [Full markdown content of page]
    </content>
  </document>
</documents>
```

The `index` attribute enables citations like "Based on Document 3..." The `source` field (URL or path) allows users to verify answers and provides the model semantic information about document hierarchy. The token overhead for this XML structure runs approximately **15% higher** than bare markdown, but the accuracy improvements—particularly for citation and attribution tasks—justify the cost.

**On Table of Contents:** A TOC at the beginning can help for very large contexts (50K+ tokens), but Google's documentation emphasizes that placing the query at the end matters more than a TOC. If you include one, place it immediately before your question as a final orientation aid, not at the beginning where it consumes privileged attention. A better alternative is the document indexing shown above, which allows natural references without separate navigation overhead.

**Flat vs. hierarchical structure** depends on your documentation's natural organization. For sites with clear parent-child page relationships, preserving hierarchy helps: `<section name="API Reference"><document>...</document><document>...</document></section>`. For flat documentation sets, simple sequential `<document>` tags suffice. Anthropic's guidance: "Nest tags for hierarchical content" but avoid excessive nesting depth.

## Gemini 2.5 Flash parameters require careful tuning

Google's default temperature of **1.0** is inappropriate for documentation Q&A. Developer community consensus and Oracle's official guidance both recommend **0.3–0.5 for factual tasks**, with some practitioners going as low as 0 for pure extraction. Higher temperatures introduce hallucinations and factually incorrect responses—exactly what you want to avoid when users trust your tool to answer questions accurately.

**Recommended parameter configuration:**

| Parameter | Recommended Value | Rationale |
|-----------|-------------------|-----------|
| Temperature | 0.3–0.5 | Reduces hallucination while maintaining natural language flow |
| Top P | 0.95 (default) | No change needed for Q&A tasks |
| Top K | 64 (fixed) | Not configurable in Gemini 2.5 Flash |
| Thinking Budget | 0 or low | Simple Q&A doesn't benefit from extended reasoning; saves tokens |

**System instruction vs. inline instruction:** Google recommends system instructions for "defining a persona or role," "output format," "style and tone," and "goals or rules." For your CLI tool, use the system instruction for persistent behavioral constraints:

```python
system_instruction = """You are a documentation assistant. Your role is to answer 
questions using ONLY the provided documentation. 

Rules:
- If the answer is not in the documentation, say "I couldn't find this information 
  in the documentation."
- Quote relevant passages when answering.
- Cite document sources using their index numbers.
- Be concise but complete."""
```

The user prompt then contains only the documents and the question, keeping the structure clean and enabling cache hits when the same documentation is queried repeatedly.

**Context caching is critical** for 100K+ token prompts with repeated documentation. Gemini 2.5 Flash supports implicit caching (automatic, 75% discount on cache hits) and explicit caching (configurable TTL, 90% discount). To maximize implicit cache hits, Google advises putting "large and common contents at the beginning of your prompt" and sending requests with similar prefixes within a short time window. Your CLI tool's architecture—same documentation, different questions—is ideal for caching. Structure prompts as: `[system instruction] + [documents] + [question]` so the first two components cache while questions vary.

## Instruction placement follows the sandwich pattern

For long-context prompts, placing instructions at **both beginning and end** counters primacy and recency effects. Google's Gemini 3 template (applicable to 2.5) explicitly includes a `<final_instruction>` section after the context. Research from Indiana University confirms this "sandwich pattern" improves instruction following for complex prompts.

**Optimal prompt structure for your CLI:**

```xml
<!-- System Instruction (separate API field) -->
You are a documentation assistant for [Library Name]. Answer questions using 
only the provided documentation. If information is not available, say so clearly.
Cite sources using document index numbers.

<!-- User Message -->
<documents>
  [Your 100K+ tokens of documentation with individual <document> wrappers]
</documents>

<question>
[User's question here]
</question>

<instructions>
Based on the documentation above, answer the question. Quote relevant passages 
to support your answer. If you cannot find the answer in the documentation, 
respond with "I couldn't find this in the documentation" rather than guessing.
</instructions>
```

The final `<instructions>` block repeats key constraints because these are most likely to be followed given recency effects. Google's guidance specifically recommends using "a clear transition phrase to bridge the context and your query, such as 'Based on the information above...'"

**Constraints that improve accuracy:**

- **"Only use the provided documentation"** — explicit grounding reduces hallucination
- **"If unsure, say so"** — Anthropic's research shows this significantly reduces confabulation
- **"Quote relevant passages first"** — forces the model to locate evidence before answering
- **"Cite your sources using document numbers"** — enables verification and improves answer quality

## Mitigation strategies for lost-in-the-middle are effective

Beyond strategic document ordering, several techniques reduce middle-position degradation:

**Quote extraction before answering** is the highest-impact technique. Anthropic's documentation recommends: "Ask the model to quote relevant parts of the documents first before carrying out its task. This helps cut through the 'noise' of the rest of the document's contents." For your CLI:

```xml
<instructions>
First, find and quote the passages from the documentation that are most relevant 
to answering this question. Place these quotes in <relevant_quotes> tags with 
their document indices.

Then, based on these quotes, provide your answer in <answer> tags.
</instructions>
```

This converts a long-context retrieval task into a focused extraction task followed by short-context reasoning, dramatically improving accuracy.

**Retrieve-then-reason** is a related pattern where the model explicitly extracts relevant content before generating an answer. GPT-4 testing showed consistent **4% improvements** using this approach. The model prepends its own extracted evidence directly before answering, effectively creating a "short-context" version of the problem.

**Document summaries before content** can help but often waste tokens. If you implement this, keep summaries to a single sentence: `<document index="1"><summary>OAuth 2.0 authentication flows and token management.</summary><content>...</content></document>`. The evidence for summaries helping is weaker than for quote extraction—use them sparingly for very large document sets where navigation is genuinely difficult.

**Repeating instructions** at multiple points in very long prompts provides "anchor points" that maintain model attention. For 100K+ token contexts, consider placing a brief reminder after every 10-15 documents: `<!-- Remember: Only answer from provided documentation. Cite sources. -->`

## Consistency and hallucination reduction require multiple approaches

Gemini 2.5 Flash has documented hallucination issues, particularly with URLs and when information isn't present in context. Community reports from Google's developer forums cite "a lot of hallucinations (especially when it comes to web URLs)" as a known issue.

**Hallucination reduction techniques:**

1. **Lower temperature** (0.3–0.5) is the single most effective lever
2. **Explicit grounding instruction**: "If the information is not available in the context, just return 'not available in the context'"
3. **Request structured output**: JSON mode constrains responses and makes hallucinations easier to detect
4. **Use quote extraction**: Forces model to cite evidence, making fabrication obvious
5. **Role assignment**: "You are a documentation expert who only states facts found in the provided documents"

**Structured output format** for maximum reliability:

```json
{
  "answer": "The authentication endpoint accepts OAuth 2.0 bearer tokens [Doc 3].",
  "sources": [
    {"document_index": 3, "quote": "Bearer tokens must be included in the Authorization header"}
  ],
  "confidence": "high",
  "found_in_docs": true
}
```

The `found_in_docs` boolean makes it trivial to detect "I couldn't find this" responses programmatically, and the required quotes force grounding.

**Known reliability issues** with Gemini 2.5 Flash include response truncation (responses stopping mid-sentence without hitting token limits—a documented P2 bug) and excessive verbosity. Implement retry logic for truncated responses and explicitly request concise answers: "Provide a focused answer in 2-3 paragraphs maximum."

## Complete recommended prompt template

Based on all research findings, here is the optimal structure for your documentation Q&A CLI:

```python
# System Instruction (separate API parameter)
system_instruction = """You are a documentation assistant for {library_name}. 
Your sole purpose is to answer questions using the provided documentation.

Rules:
1. ONLY use information from the provided documents
2. If information isn't in the documents, say "I couldn't find this in the documentation"
3. Always cite sources using document index numbers like [Doc 3]
4. Quote relevant passages to support your answers
5. Be concise and direct"""

# User Message Template
user_message = """<documents>
{documents_xml}
</documents>

<question>
{user_question}
</question>

<instructions>
First, identify and quote the most relevant passages from the documentation above.
Then provide your answer based on these passages.

Format your response as:
<relevant_quotes>
[Quoted passages with document indices]
</relevant_quotes>

<answer>
[Your answer with inline citations like [Doc 1]]
</answer>
</instructions>"""
```

**Configuration:**
```python
generation_config = {
    "temperature": 0.4,
    "top_p": 0.95,
    "max_output_tokens": 4096,
    "thinking_budget": 0  # Disable extended thinking for simple Q&A
}
```

**Key trade-offs to consider:**

| Approach | Tokens | Accuracy | Latency |
|----------|--------|----------|---------|
| XML wrappers + metadata | +15% | Higher | Minimal impact |
| Quote extraction first | +20-30% output | Significantly higher | Higher |
| Document summaries | +10-20% | Marginal improvement | Minimal |
| Lower temperature | Same | Higher factual accuracy | Same |
| Repeated instructions | +1-2% | Better instruction following | Same |

For your 100K+ token documentation use case, prioritize: XML structure, query-at-end placement, quote extraction, and low temperature. Skip document summaries unless navigation becomes problematic. Enable context caching to reduce costs by 75-90% for repeated documentation sets.

## Conclusion

The optimal prompt structure for Gemini 2.5 Flash documentation Q&A combines several evidence-based strategies: **XML document wrappers** with source metadata for clear semantic boundaries, **relevance-based ordering** with most important documents near the query, **sandwich-pattern instruction placement** with constraints at both beginning and end, and **quote extraction** before answering to force grounding. Temperature should be reduced to **0.3–0.5** from the default 1.0, and explicit "say I don't know" instructions significantly reduce hallucination.

Gemini 2.5 Flash's 1M-token context and strong long-context performance make it well-suited for documentation Q&A without RAG, but the model still exhibits position bias and hallucination tendencies that require mitigation. Context caching is essential for cost management at scale—structure prompts with static documentation first to maximize cache hits. The quote-extraction pattern represents the highest-impact accuracy improvement for this use case, converting long-context retrieval into a two-stage process that leverages the model's strengths while compensating for its weaknesses.
