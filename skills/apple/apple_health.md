---
name: apple_health
description: "Apple Health data access"
version: "1.0"
trigger: "apple health fitness data"
platforms: ["macos", "ios"]
requires_tools: ["run_command"]
---

# Apple Health data access

## Purpose
Apple Health data access through the Hera agent interface, enabling hands-free control and automation of Apple ecosystem features.

## Instructions
1. Parse the user's intent regarding apple health functionality
2. Determine the appropriate Apple API or AppleScript command
3. Execute the operation with proper error handling
4. Confirm the action to the user with relevant details
5. Suggest related follow-up actions

## Capabilities
- Query current state and retrieve information
- Create, modify, and delete items as requested
- Search and filter by various criteria
- Integrate with other Apple services and Shortcuts
- Handle authorization and permission requirements

## Integration
- Works with Siri Shortcuts for complex automations
- Supports AppleScript for macOS automation
- Uses Apple's Frameworks via command-line bridges
- Respects user privacy and permission boundaries

## Error Handling
- Check for required permissions before attempting operations
- Provide clear feedback when operations are not available
- Suggest alternative approaches when primary method fails
- Handle network-dependent features gracefully when offline

## Best Practices
- Always confirm destructive operations before executing
- Use native Apple APIs when available for best compatibility
- Cache frequently accessed data to reduce latency
- Respect system resource limits and battery considerations
