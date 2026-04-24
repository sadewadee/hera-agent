---
name: task_decompose
description: Break down complex tasks into smaller, actionable steps
triggers:
  - plan
  - break down
  - decompose
  - step by step
  - how to
platforms: []
requires_tools: []
---

# Task Decomposition

When the user presents a complex task, break it down into manageable steps before executing.

## Process

### 1. Understand the Goal
- What is the desired end state?
- What are the constraints (time, resources, technology)?
- Are there dependencies or prerequisites?

### 2. Identify Major Phases
Break the task into 3-7 major phases. Each phase should:
- Have a clear objective
- Be completable independently (where possible)
- Have a verifiable outcome

### 3. Decompose Each Phase
For each phase, list the specific steps:
- What action to take
- What tools or resources are needed
- What the expected output is
- How to verify success

### 4. Identify Risks and Dependencies
- Which steps depend on others?
- What could go wrong at each step?
- Are there alternative approaches if a step fails?

## Output Format

```
## Task Plan: [task name]

### Overview
[1-2 sentence summary of the task and approach]

### Phase 1: [phase name]
Objective: [what this phase achieves]
Steps:
1. [step] - [expected outcome]
2. [step] - [expected outcome]
Verification: [how to confirm this phase is complete]

### Phase 2: [phase name]
...

### Risks and Mitigations
- Risk: [description] -> Mitigation: [approach]

### Estimated Effort
[rough estimate of time or complexity]
```

## Guidelines

- Start with the end in mind: define success criteria first
- Prefer smaller steps that can be verified independently
- Make each step actionable (starts with a verb)
- Include verification criteria for each phase
- Be honest about uncertainty: flag steps where you are unsure
- Ask the user for clarification before starting if requirements are ambiguous
