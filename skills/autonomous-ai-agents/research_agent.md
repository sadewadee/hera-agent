---
name: research_agent
description: "Autonomous research and information gathering agents"
version: "1.0"
trigger: "research agent autonomous gathering"
platforms: []
requires_tools: ["web_search"]
---

# Autonomous research and information gathering agents

## Purpose
Autonomous research and information gathering agents for building and orchestrating autonomous AI agent systems.

## Instructions
1. Analyze the agent architecture requirements
2. Design the agent pipeline with appropriate components
3. Implement with proper error handling and fallback strategies
4. Test agent behavior across diverse scenarios
5. Monitor agent performance and iterate on design

## Core Concepts
- Agent loop: Observe -> Think -> Act -> Evaluate
- Tool integration for extending agent capabilities
- Memory management for maintaining context across interactions
- Planning for multi-step task decomposition
- Safety constraints to prevent harmful or unintended actions

## Architecture Patterns
- **ReAct**: Reasoning + Acting in interleaved steps
- **Chain-of-Thought**: Step-by-step reasoning before action
- **Tree-of-Thought**: Explore multiple reasoning paths
- **Plan-and-Execute**: Generate full plan then execute steps
- **Reflexion**: Learn from past mistakes via self-reflection

## Implementation Considerations
- Define clear success/failure criteria for agent tasks
- Implement timeout and retry mechanisms
- Log all agent decisions and actions for debugging
- Use structured output parsing for reliable tool calls
- Set resource limits (API calls, tokens, time) to prevent runaway agents

## Evaluation Metrics
- Task completion rate across test scenarios
- Average steps to completion (efficiency)
- Error recovery rate (resilience)
- Cost per task (API calls, tokens used)
- Safety violation rate (harmful actions attempted)

## Best Practices
- Start simple and add complexity incrementally
- Test with adversarial and edge-case inputs
- Implement graceful degradation when tools fail
- Use human-in-the-loop for high-stakes decisions
- Version control agent prompts and configurations
