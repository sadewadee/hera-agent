---
name: c4_model
description: "C4 model architecture diagrams"
version: "1.0"
trigger: "c4 model architecture diagram context container"
platforms: []
requires_tools: []
---

# C4 Model Diagrams

## Purpose
Create architecture diagrams using the C4 model (Context, Container, Component, Code) for clear system documentation.

## Instructions
1. Start with Level 1 (System Context) diagram
2. Zoom into Level 2 (Container) for the system
3. Detail Level 3 (Component) for specific containers
4. Use Level 4 (Code) only when needed for critical components
5. Keep diagrams current as architecture evolves

## Levels
- **Context (L1)**: System and its interactions with users and external systems
- **Container (L2)**: High-level technology choices (web app, API, database)
- **Component (L3)**: Components within a container (controllers, services, repos)
- **Code (L4)**: Class/function level detail (usually auto-generated)

## Structurizr DSL
```
workspace {
    model {
        user = person "User"
        system = softwareSystem "My System" {
            webapp = container "Web App" "React SPA" "JavaScript"
            api = container "API" "REST API" "Go"
            db = container "Database" "PostgreSQL"
        }
        user -> webapp "Uses"
        webapp -> api "Calls"
        api -> db "Reads/Writes"
    }
}
```

## Best Practices
- Start at Level 1 and zoom in as needed
- Keep each diagram focused and readable
- Include a key/legend for notation
- Update diagrams when architecture changes
- Use consistent notation across all levels
