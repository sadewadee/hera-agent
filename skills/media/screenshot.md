---
name: screenshot
description: "Screenshot capture and annotation"
version: "1.0"
trigger: "screenshot capture screen annotation"
platforms: []
requires_tools: ["run_command"]
---

# Screenshot Capture

## Purpose
Capture, annotate, and share screenshots for documentation and communication.

## Instructions
1. Capture the desired screen area
2. Annotate with highlights, arrows, or text
3. Optimize file size for sharing
4. Save or share in the appropriate format
5. Organize screenshots for reference

## macOS Commands
```bash
# Full screen
screencapture screen.png

# Selection
screencapture -s selection.png

# Window
screencapture -w window.png

# Clipboard
screencapture -c
```

## Annotation Tools
- macOS Preview for basic markup
- Shottr for advanced annotations
- CleanShot X for professional screenshots
- ImageMagick for CLI-based processing

## Best Practices
- Crop to relevant area only
- Use consistent annotation styles
- Redact sensitive information
- Use PNG for UI screenshots, JPEG for photos
- Organize with descriptive filenames
