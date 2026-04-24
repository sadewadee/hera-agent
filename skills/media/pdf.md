---
name: pdf
description: "PDF creation, editing, and extraction"
version: "1.0"
trigger: "pdf create edit extract convert"
platforms: []
requires_tools: ["run_command"]
---

# PDF Processing

## Purpose
Create, edit, and extract content from PDF documents.

## Instructions
1. Identify the PDF operation needed
2. Select appropriate tools for the task
3. Process the document with error handling
4. Validate the output
5. Deliver in the requested format

## Operations
- Create PDFs from text, HTML, or images
- Extract text and metadata from PDFs
- Merge and split PDF files
- Convert between PDF and other formats
- Fill PDF forms programmatically

## Tools
- `wkhtmltopdf`: HTML to PDF conversion
- `pdftotext`: Text extraction from PDFs
- `qpdf`: PDF manipulation and merging
- `pdftk`: PDF toolkit for forms and metadata
- Python: PyPDF2, reportlab, pdfplumber

## Best Practices
- Preserve formatting during conversion
- Handle multi-page and large documents efficiently
- Validate output before delivery
- Handle encrypted PDFs appropriately
- Clean up temporary files after processing
