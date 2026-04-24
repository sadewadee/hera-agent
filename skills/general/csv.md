---
name: csv
description: "CSV file processing and analysis"
version: "1.0"
trigger: "csv parse process data spreadsheet"
platforms: []
requires_tools: ["run_command"]
---

# CSV Processing

## Purpose
Parse, analyze, and transform CSV files using command-line tools and programming libraries.

## Instructions
1. Inspect CSV structure (headers, delimiters, encoding)
2. Clean and validate data
3. Apply transformations and aggregations
4. Export in the required format
5. Handle edge cases (quotes, newlines, encoding)

## Command-Line Tools
- `csvkit`: Swiss army knife for CSV processing
- `xsv`: Fast CSV toolkit written in Rust
- `miller`: Like awk for CSV, TSV, JSON
- `cut`, `sort`, `uniq`: Unix basics for simple operations

## Best Practices
- Always check encoding (UTF-8 preferred)
- Handle quoted fields with embedded delimiters
- Validate data types in each column
- Use streaming for large files
- Document any transformations applied
