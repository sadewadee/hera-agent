---
name: index-cache
description: Cached skill index data from external skill registries for offline discovery and search
version: 1.0.0
metadata:
  hera:
    tags: [meta, skills, index, cache]
    related_skills: []
---

# Index Cache

This directory contains cached copies of external skill registry indices.
These caches enable offline skill discovery and search without requiring
network access to the upstream registries.

## Files

| File | Source | Description |
|------|--------|-------------|
| `anthropics_skills_skills_.json` | Anthropic | Anthropic's skill registry |
| `claude_marketplace_anthropics_skills.json` | Claude Marketplace | Claude marketplace skills |
| `lobehub_index.json` | LobeHub | LobeHub plugin/skill index |
| `openai_skills_skills_.json` | OpenAI | OpenAI's skill/plugin registry |

## Refresh

These caches are automatically refreshed when you run:

```bash
hera skills refresh-index
```

Or manually fetch a specific registry:

```bash
hera skills refresh-index --source anthropic
hera skills refresh-index --source lobehub
```

## Usage

The cached indices are used by:
- `hera skills search <query>` -- searches across all cached registries
- `hera skills browse` -- lists available skills from all sources
- `hera skills install <name>` -- resolves skill metadata from cache

## Notes

- Cache files are excluded from the Nix package build (see `nix/packages.nix`)
- Stale caches are refreshed automatically if older than 24 hours
- Network failures gracefully fall back to the most recent cache
