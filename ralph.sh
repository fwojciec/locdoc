#!/bin/bash
# Ralph Wiggum Loop - autonomous task execution for an epic
#
# Usage: ./ralph.sh <epic-id>
# Example: ./ralph.sh locdoc-80r

set -e

EPIC="${1:-locdoc-80r}"
CLAUDE="${CLAUDE:-$HOME/.claude/local/claude}"

echo "🔄 Starting Ralph loop for epic: $EPIC"

while :; do
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🔍 Looking for next ready task..."

    "$CLAUDE" -p "/ralph-iterate $EPIC" --dangerously-skip-permissions

    # Check if epic is complete
    if [ -f .ralph-complete ]; then
        rm .ralph-complete
        echo ""
        echo "✅ Epic $EPIC complete!"
        break
    fi

    # Small delay between iterations
    sleep 2
done
