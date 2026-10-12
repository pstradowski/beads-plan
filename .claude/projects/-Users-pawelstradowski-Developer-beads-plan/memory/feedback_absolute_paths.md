---
name: Use absolute file paths
description: Always use absolute paths when referencing files so iTerm2 cmd-click works correctly
type: feedback
---

Always use absolute file paths (e.g., `/Users/pawelstradowski/Developer/beads-plan/openspec/changes/...`) instead of relative paths when referencing files in output.

**Why:** Relative paths like `openspec/changes/foo/` get interpreted as HTTP URLs by iTerm2 on cmd-click instead of opening the local file.

**How to apply:** Whenever mentioning a file or directory path in text output, use the full absolute path.
