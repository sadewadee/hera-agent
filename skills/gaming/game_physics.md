---
name: game_physics
description: "Game physics simulation and implementation"
version: "1.0"
trigger: "game physics simulation collision rigid body"
platforms: []
requires_tools: ["run_command"]
---

# Game Physics

## Purpose
Implement physics simulation for games including collision detection, rigid body dynamics, and particle systems.

## Instructions
1. Define physics requirements (2D vs 3D, accuracy vs performance)
2. Choose physics engine or implement custom solution
3. Configure collision layers and detection
4. Implement physics-based gameplay mechanics
5. Optimize for target frame rate

## Physics Engines
- **Box2D**: 2D physics, widely used in indie games
- **Bullet**: 3D physics, open source, used in many AAA games
- **PhysX**: NVIDIA, integrated in Unity and Unreal
- **Jolt**: Modern C++, high performance
- **Rapier**: Rust-based, 2D/3D, WebAssembly compatible

## Key Concepts
- Rigid body dynamics (mass, velocity, forces, torques)
- Collision detection (broad phase, narrow phase)
- Collision response (impulse resolution, friction)
- Constraints and joints (hinges, springs, sliders)
- Continuous collision detection (CCD) for fast objects

## Performance
- Use spatial partitioning (quad-tree, octree, sweep-and-prune)
- Sleep inactive bodies to save computation
- Reduce collision geometry complexity
- Fixed timestep for deterministic simulation
- Profile physics step cost per frame

## Best Practices
- Use a fixed timestep with interpolation for rendering
- Separate physics layers for different interaction types
- Use simplified collision shapes (not visual mesh)
- Test with extreme values to find stability limits
- Profile before optimizing
