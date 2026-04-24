# Skills

Skills are Markdown files with YAML frontmatter that give the agent specialized instructions for specific tasks. Hera ships with 170+ bundled skills and supports user-created custom skills.

## What is a skill?

A skill is a `.md` file with a YAML header followed by instruction content:

```markdown
---
name: code-reviewer
description: Review code for bugs, style issues, and best practices
triggers:
  - review this code
  - code review
  - check my code
requires_tools:
  - read_file
platforms:
  - cli
  - telegram
---

When asked to review code:

1. Read the file contents using read_file
2. Check for: logic errors, edge cases, error handling, style consistency
3. Format feedback as: [ISSUE TYPE] line N: description
4. Summarize findings with a severity count
```

## Frontmatter fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Unique identifier for the skill |
| `description` | string | What the skill does (shown in skill list) |
| `triggers` | string[] | Phrases that activate this skill |
| `requires_tools` | string[] | Tools that must be available |
| `platforms` | string[] | Platforms where this skill is active (empty = all) |

## Skill tiers

| Tier | Location | Description |
|------|----------|-------------|
| `bundled` | `skills/` in repo | Ships with Hera |
| `optional` | `skills/optional/` | Available but not loaded by default |
| `user` | `~/.hera/skills/` | Your custom skills |

## Skill directories

Hera loads skills from multiple directories at startup:

```
skills/general/     — general-purpose skills
skills/system/      — system administration skills
skills/optional/    — optional community skills
~/.hera/skills/     — your custom skills
```

## Skill nudges

Every `agent.skill_nudge_interval` turns (default: 15), the agent is reminded of available skills. It can then proactively suggest or apply a relevant skill.

```yaml
agent:
  skill_nudge_interval: 15
```

## Using skills

Skills activate when their trigger phrases appear in a message. You can also use the `/` command syntax:

```
You: /code-reviewer
You: review this code for me
```

The agent can also list available skills:

```
You: What skills do you have?

Hera: [using skills tool: list]
Available skills:
- code-reviewer: Review code for bugs and style issues
- git-helper: Git workflow assistance
- research: Deep research with web search
...
```

## Installing community skills

The `skills` tool can install skills from GitHub:

```
You: Install the "docker-expert" skill from the community hub

Hera: [using skills_sync tool]
Installing docker-expert...done. Skill available as /docker-expert
```

Skills directories in the repo are organized by category:

```
skills/
  apple/
  creative/
  data-science/
  devops/
  domain/
  email/
  feeds/
  gaming/
  general/
  github/
  inference-sh/
  leisure/
  media/
  mlops/
  productivity/
  research/
  social-media/
  software-development/
  system/
```

## Writing custom skills

Create a file at `~/.hera/skills/my-skill.md`:

```markdown
---
name: my-skill
description: Does something custom for my workflow
triggers:
  - my custom task
  - run my workflow
---

Instructions for the agent go here in plain Markdown.

## Steps

1. First do this
2. Then do that
3. Use the shell tool to run: ./my-script.sh

## Output format

Always respond with:
- Summary: one line
- Details: bulleted list
```

Hera picks up new skill files automatically on the next run (or restart).
