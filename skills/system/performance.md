---
name: performance
description: "System performance monitoring and tuning"
version: "1.0"
trigger: "system performance monitoring cpu memory"
platforms: []
requires_tools: ["run_command"]
---

# System Performance

## Purpose
Monitor, analyze, and optimize system performance including CPU, memory, disk, and network.

## Instructions
1. Establish baseline performance metrics
2. Monitor key indicators continuously
3. Identify bottlenecks and resource constraints
4. Apply tuning and optimization
5. Verify improvement and document changes

## Key Tools
- `top`/`htop`: Real-time process monitoring
- `vmstat`: Virtual memory statistics
- `iostat`: Disk I/O statistics
- `netstat`/`ss`: Network connections
- `sar`: Historical system activity

## Common Bottlenecks
- CPU: High utilization, context switching
- Memory: Swapping, high memory pressure
- Disk: I/O wait, queue depth
- Network: Bandwidth saturation, connection limits

## Best Practices
- Monitor before optimizing (measure first)
- Address the biggest bottleneck first
- Make one change at a time
- Verify improvement with metrics
- Document all tuning changes
