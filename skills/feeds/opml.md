---
name: opml
description: "OPML feed list import and export"
version: "1.0"
trigger: "opml feed list import export"
platforms: []
requires_tools: ["run_command"]
---

# OPML Management

## Purpose
Import and export feed subscriptions using OPML format for portability across feed readers.

## Instructions
1. Parse OPML files for feed URLs and metadata
2. Organize feeds into categories from outline structure
3. Export current subscriptions as valid OPML
4. Merge OPML files from multiple sources
5. Validate OPML structure and feed URLs

## OPML Format
```xml
<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>My Feeds</title></head>
  <body>
    <outline text="Tech" title="Tech">
      <outline type="rss" text="Hacker News" xmlUrl="https://hn.algolia.com/rss"/>
    </outline>
  </body>
</opml>
```

## Best Practices
- Validate all feed URLs before import
- Organize feeds into meaningful categories
- Remove duplicate subscriptions during merge
- Back up OPML before making changes
- Include feed titles and categories for clarity
