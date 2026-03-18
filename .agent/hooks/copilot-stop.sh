#!/usr/bin/env sh
# Copilot Stop hook
# Runs when an agent session ends.
# Ensures bd data is pushed and work is properly closed.

# Remind to push if there are uncommitted beads changes
bd dolt status 2>/dev/null | grep -q 'modified\|new file' && \
  printf '{"systemMessage": "Reminder: run `bd dolt push` and `git push` before ending the session."}\n' || true

exit 0
