---
name: literature_review
description: "Systematic ML research literature review"
version: "1.0"
trigger: "literature review papers survey research"
platforms: []
requires_tools: ["web_search"]
---

# Literature Review

## Purpose
Conduct systematic literature reviews of ML research papers, identifying key contributions, methods, and trends in a specific area.

## Instructions
1. Define the research question and scope
2. Search across key venues (arXiv, NeurIPS, ICML, ICLR, ACL, CVPR)
3. Filter papers by relevance, recency, and citation count
4. Categorize papers by methodology and contribution type
5. Synthesize findings into a structured summary

## Search Strategy
- Start with recent survey papers in the area
- Use Semantic Scholar API for citation graph traversal
- Search arXiv with category filters (cs.LG, cs.CL, cs.CV, stat.ML)
- Check proceedings of top venues from last 2-3 years
- Follow citation chains from seminal papers

## Paper Categorization
- **Methodology**: New architecture, training technique, loss function
- **Application**: Novel application of existing methods
- **Empirical study**: Benchmarking, ablation, comparison
- **Theoretical**: Proofs, bounds, convergence analysis
- **System**: Infrastructure, efficiency, scaling

## Summary Template
```markdown
## Paper: [Title]
- **Authors**: [names], **Year**: [year], **Venue**: [venue]
- **Key contribution**: One sentence summary
- **Method**: What they did technically
- **Results**: Main empirical findings
- **Limitations**: What they didn't address
- **Relevance**: How this relates to our work
```

## Tools
- Semantic Scholar API for automated paper search and citation data
- Connected Papers for visual citation graph exploration
- Papers With Code for reproducibility and benchmark results
- Zotero/Mendeley for reference management

## Best Practices
- Read abstracts first to filter, then introduction/conclusion, then full paper
- Track which papers you've read and your notes
- Identify disagreements between papers and investigate
- Note reproducibility information (code available, datasets public)
