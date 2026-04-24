---
name: json
description: "JSON processing, validation, and transformation"
version: "1.0"
trigger: "json parse validate transform jq"
platforms: []
requires_tools: ["run_command"]
---

# JSON Processing

## Purpose
Parse, validate, transform, and query JSON data using command-line and programming tools.

## Instructions
1. Validate JSON structure and syntax
2. Query specific fields with jq or similar tools
3. Transform between JSON and other formats
4. Merge, filter, and reshape JSON documents
5. Pretty-print for readability

## jq Examples
```bash
# Extract a field
echo '{"name":"Alice","age":30}' | jq '.name'

# Filter arrays
echo '[{"status":"active"},{"status":"inactive"}]' | jq '.[] | select(.status=="active")'

# Transform shape
echo '{"users":[{"name":"Alice"},{"name":"Bob"}]}' | jq '.users[].name'
```

## Best Practices
- Validate JSON before processing
- Use schema validation for API inputs/outputs
- Handle null and missing fields gracefully
- Use streaming parsers for large files
- Pretty-print for human readability, compact for storage/transfer
