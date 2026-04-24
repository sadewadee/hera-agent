---
name: network_diagram
description: "Network topology diagrams"
version: "1.0"
trigger: "network diagram topology infrastructure"
platforms: []
requires_tools: []
---

# Network topology diagrams

## Purpose
Network topology diagrams using text-based diagramming tools (Mermaid, PlantUML, D2) for documentation and communication.

## Instructions
1. Identify the system or process to diagram
2. Determine the appropriate diagram type
3. Gather components, relationships, and flows
4. Generate diagram code in the selected format
5. Iterate on layout and clarity

## Mermaid Syntax
- Use Mermaid for quick, Git-friendly diagrams
- Supported in GitHub, GitLab, Notion, and many tools
- Render with mermaid-cli or browser-based editors

## PlantUML Alternative
- More expressive for complex UML diagrams
- Requires Java runtime for rendering
- Better for formal software architecture documentation

## D2 Alternative
- Modern, declarative diagram language
- Built-in themes and auto-layout
- Good for infrastructure and system diagrams

## Design Principles
- Keep diagrams focused on one concern
- Use consistent naming conventions
- Add labels to relationships and flows
- Group related components visually
- Include a legend for non-obvious symbols

## Best Practices
- Store diagram source in version control alongside code
- Update diagrams when architecture changes
- Use automated rendering in CI/CD pipelines
- Keep diagrams at the right abstraction level
- Link diagrams to relevant documentation
