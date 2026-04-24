---
name: gif_optimize
description: "GIF optimization and compression"
version: "1.0"
trigger: "gif optimize compress size reduce"
platforms: []
requires_tools: ["run_command"]
---

# GIF Optimization

## Purpose
Optimize and compress GIF files for web delivery without significant quality loss.

## Instructions
1. Analyze the GIF for optimization opportunities
2. Reduce color palette to minimum needed
3. Optimize frame disposal and transparency
4. Reduce dimensions if appropriate
5. Apply lossy compression if acceptable

## Techniques
- Reduce color palette (256 -> 128 -> 64 colors)
- Crop to remove unnecessary areas
- Reduce frame rate (skip frames)
- Apply lossy compression with gifsicle
- Convert to WebM/MP4 for even smaller files

## Tools
- `gifsicle`: GIF optimization CLI tool
- `ffmpeg`: Convert GIF to video formats
- `ImageMagick`: Resize and manipulate GIFs

## Best Practices
- Target under 5MB for web/chat use
- Consider WebM/MP4 as smaller alternatives
- Preview quality after compression
- Keep originals before optimizing
- Test across platforms for compatibility
