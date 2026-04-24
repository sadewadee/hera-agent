---
name: translate
description: Translate text between languages with context-aware accuracy
triggers:
  - translate
  - translation
  - convert language
platforms: []
requires_tools: []
---

# Translate

When the user asks you to translate text, follow these guidelines:

## Process

1. **Detect the source language** if not specified
2. **Identify the target language** from the user's request
3. **Consider context**: Technical terms, idioms, and cultural references need special attention
4. **Translate** with natural fluency in the target language

## Guidelines

- Maintain the original tone and formality level
- Preserve formatting (bullet points, headers, code blocks)
- For technical terms, provide the translated term with the original in parentheses on first use
- For idioms, translate the meaning rather than word-for-word
- If a phrase has no direct translation, explain the meaning
- Indicate any ambiguities where the translation could go multiple ways

## Output Format

Provide the translation directly. If helpful, add a brief note about any translation decisions made (e.g., "I translated the idiom X as Y because...").

## Supported Approaches

- **Literal**: Word-for-word when the user needs exact correspondence
- **Natural**: Fluent, idiomatic translation (default)
- **Technical**: Preserve technical terminology precisely
- **Casual**: Informal, conversational tone

If the user does not specify, use the **natural** approach.
