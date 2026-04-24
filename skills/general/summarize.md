---
name: summarize
description: Summarize text, articles, or conversations into concise key points
triggers:
  - summarize
  - summary
  - tldr
  - recap
platforms: []
requires_tools: []
---

# Summarize

When the user asks you to summarize content, follow these guidelines:

## Process

1. **Identify the content type**: Is it a conversation, article, code, document, or raw text?
2. **Determine the desired length**: Brief (1-2 sentences), standard (3-5 bullet points), or detailed (full paragraph summary)
3. **Extract key points**: Focus on the most important information, decisions, and action items

## Output Format

### Brief Summary
Provide a 1-2 sentence overview capturing the essence of the content.

### Standard Summary
- Use bullet points for key points
- Include any decisions made or conclusions reached
- Note any action items or next steps
- Keep each point concise (one line)

### Detailed Summary
Provide a structured summary with:
- **Context**: What is the content about?
- **Key Points**: The main takeaways
- **Decisions**: Any decisions or conclusions
- **Action Items**: Next steps or tasks
- **Notable Details**: Important specifics worth preserving

## Guidelines

- Preserve factual accuracy; do not add information not present in the source
- Use neutral language unless the user requests a specific tone
- If summarizing code, focus on what it does, not line-by-line explanation
- For conversations, capture the outcome and any agreed-upon actions
- If the content is in a different language, summarize in the user's language unless asked otherwise
