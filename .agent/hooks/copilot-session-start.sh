#!/usr/bin/env sh
# Copilot SessionStart hook
# Runs once at the beginning of each agent session.
# Primes bd with ready tasks and injects project context into the session.

bd prime 2>/dev/null || true

# Output a system message with project context so Copilot starts informed
cat <<'EOF'
{
  "systemMessage": "Project uses bd (beads) for issue tracking. Run `bd prime` to make sure understand how to use bd. Run `bd ready` to find available work."
}
EOF
