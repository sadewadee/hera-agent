---
name: code_review
description: Review code for quality, bugs, security issues, and improvement opportunities
triggers:
  - review
  - code review
  - review code
  - check code
platforms: []
requires_tools:
  - file_read
---

# Code Review

When the user asks you to review code, follow this structured process:

## Review Checklist

### 1. Correctness
- Does the code do what it claims to do?
- Are there off-by-one errors, null pointer risks, or unhandled edge cases?
- Are return values and error codes checked?

### 2. Security
- Is user input validated and sanitized?
- Are there SQL injection, XSS, or command injection risks?
- Are secrets hardcoded?
- Are permissions checked where needed?

### 3. Performance
- Are there unnecessary allocations or copies?
- Are there N+1 query patterns?
- Could any loops be replaced with more efficient approaches?
- Are there missing indexes implied by the query patterns?

### 4. Readability
- Are names clear and descriptive?
- Is the code well-structured and modular?
- Are complex sections commented?
- Is there unnecessary complexity?

### 5. Error Handling
- Are errors propagated with context?
- Are errors logged at the appropriate level?
- Is cleanup (defer, finally) handled correctly?

### 6. Testing
- Is the code testable?
- Are edge cases covered?
- Are there missing test scenarios?

## Output Format

```
## Code Review: [file or component name]

### Summary
[1-2 sentence overall assessment]

### Issues Found

#### Critical (must fix)
- [issue with file:line reference]

#### Important (should fix)
- [issue with file:line reference]

#### Suggestion (nice to have)
- [improvement idea]

### Positive Notes
- [what the code does well]
```

## Guidelines

- Be specific: reference file names and line numbers
- Be constructive: suggest fixes, not just problems
- Prioritize: critical issues first, style nits last
- Acknowledge good patterns and practices
- Consider the project's existing style and conventions
