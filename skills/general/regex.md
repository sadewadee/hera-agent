---
name: regex
description: "Regular expression crafting and debugging"
version: "1.0"
trigger: "regex regular expression pattern match"
platforms: []
requires_tools: []
---

# Regular Expressions

## Purpose
Create, debug, and optimize regular expressions for text matching, extraction, and validation.

## Instructions
1. Understand the text pattern to match
2. Build the regex incrementally, testing each component
3. Consider edge cases and invalid inputs
4. Optimize for readability and performance
5. Add comments and documentation

## Common Patterns
- Email: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
- URL: `https?://[^\s]+`
- Phone: `\+?[\d\s\-()]{7,15}`
- Date (ISO): `\d{4}-\d{2}-\d{2}`
- IP address: `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`

## Regex Flags
- `i` - Case insensitive
- `g` - Global (all matches)
- `m` - Multiline (^ and $ match line boundaries)
- `s` - Dotall (. matches newlines)
- `x` - Extended (allow whitespace and comments)

## Performance Tips
- Avoid catastrophic backtracking (nested quantifiers)
- Use atomic groups or possessive quantifiers when available
- Prefer specific character classes over `.`
- Anchor patterns when possible (^, $, \b)
- Test with large inputs for performance

## Best Practices
- Build and test incrementally
- Use named capture groups for readability
- Document complex patterns with comments
- Test with both matching and non-matching inputs
- Consider using a regex library for complex parsing
