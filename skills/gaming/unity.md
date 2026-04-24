---
name: unity
description: "Unity game development"
version: "1.0"
trigger: "unity game engine c# development"
platforms: []
requires_tools: ["run_command"]
---

# Unity game development

## Purpose
Unity game development with focus on practical implementation patterns and industry best practices.

## Instructions
1. Define the game requirements and target platform
2. Design the architecture and core systems
3. Implement with appropriate engine/framework patterns
4. Test across target platforms and configurations
5. Optimize for performance and player experience

## Core Systems
- Game loop and update cycle management
- Input handling and player controls
- Physics and collision detection
- Rendering pipeline optimization
- Audio system and spatial sound

## Design Patterns
- **Entity-Component-System (ECS)**: Data-oriented architecture for performance
- **State machines**: For game states, AI behavior, animations
- **Observer pattern**: For event-driven game systems
- **Object pooling**: For frequently created/destroyed objects
- **Command pattern**: For input handling and replay systems

## Performance Considerations
- Target frame rate and frame budget allocation
- Memory management and garbage collection
- Asset loading and streaming
- Level of detail (LOD) systems
- Profiling tools and optimization techniques

## Testing
- Unit tests for game logic systems
- Automated playtesting for regression detection
- Performance profiling across target hardware
- Multiplayer testing with simulated latency
- Platform-specific compliance testing

## Best Practices
- Separate game logic from rendering and input
- Use data-driven design for content and configuration
- Implement proper save/load systems early
- Profile before optimizing (avoid premature optimization)
- Plan for localization from the start
