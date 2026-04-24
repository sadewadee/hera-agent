---
name: crewai
description: "CrewAI agent orchestration"
version: "1.0"
trigger: "crewai crew agent role task"
platforms: []
requires_tools: ["run_command"]
---

# CrewAI agent orchestration

## Purpose
CrewAI agent orchestration for building autonomous AI agent systems with specific frameworks and tools.

## Instructions
1. Set up the framework environment and dependencies
2. Define agent capabilities and tool integrations
3. Configure the agent's reasoning and planning strategy
4. Implement error handling and safety guardrails
5. Test across diverse scenarios and edge cases

## Framework Setup
- Install dependencies and configure API keys
- Initialize the agent with appropriate LLM backend
- Register tools and capabilities
- Configure memory and context management
- Set up logging and monitoring

## Agent Configuration
- Define the agent's role and objectives
- Configure tool selection and usage policies
- Set up memory (short-term and long-term)
- Define planning strategy (ReAct, CoT, etc.)
- Implement safety constraints and guardrails

## Tool Integration
- Define tool schemas with clear descriptions
- Handle tool execution errors gracefully
- Implement retry logic for transient failures
- Validate tool outputs before using in reasoning
- Log all tool invocations for debugging

## Testing
- Test with diverse input scenarios
- Verify tool selection accuracy
- Check error handling and recovery
- Measure task completion rates
- Profile cost and latency per task

## Best Practices
- Use structured output parsing for reliability
- Implement cost controls (token limits, API call limits)
- Monitor agent behavior in production
- Version control agent configurations
- Use evaluation suites for regression testing
