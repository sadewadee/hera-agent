---
name: self_improve
description: Analyze completed tasks and create reusable skills for future use
triggers:
  - self improve
  - learn from this
  - create skill from this
platforms: []
requires_tools:
  - skill_create
  - memory_save
---

# Self-Improvement

After completing a complex or multi-step task, use this skill to capture what you learned for future reuse.

## When to Self-Improve

Trigger self-improvement when:
- You completed a multi-step task that required specific knowledge
- You discovered a non-obvious approach to solve a problem
- The user corrected you and the correction has general applicability
- You used a specific tool chain that worked well together

## Process

### 1. Analyze What Was Done
- What was the task?
- What steps did you take?
- What tools did you use and in what order?
- What was the outcome?

### 2. Extract Reusable Knowledge
- What would be useful to know next time?
- Are there patterns or procedures worth saving?
- Were there pitfalls to avoid?

### 3. Decide: Skill or Memory?

**Create a Skill** when:
- The knowledge is procedural (step-by-step instructions)
- It applies to a category of tasks, not just one instance
- It involves specific tool usage patterns

**Save to Memory** when:
- The knowledge is factual (a preference, a configuration detail)
- It is specific to this user or project
- It is a single piece of information

### 4. Create the Artifact

For skills, use the `skill_create` tool with:
- A clear, descriptive name
- Relevant trigger keywords
- Step-by-step instructions in the content

For memories, use the `memory_save` tool with:
- A descriptive key
- The full fact as the value

## Guidelines

- Do not create skills for trivial tasks
- Keep skill instructions concise and actionable
- Include examples where they add clarity
- Test the skill mentally: would following these instructions produce the right result?
