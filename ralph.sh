#!/bin/bash
# Ralph Wiggum Loop - autonomous task execution for an epic
#
# Usage: ./ralph.sh <epic-id>
# Example: ./ralph.sh locdoc-80r

set -e

EPIC="${1:-locdoc-80r}"
CLAUDE="${CLAUDE:-$HOME/.claude/local/claude}"
LOGFILE="ralph-$(date +%Y%m%d-%H%M%S).log"

echo "🔄 Starting Ralph loop for epic: $EPIC"
echo "📝 Logging to: $LOGFILE"

while :; do
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🔍 Looking for next ready task..."

    "$CLAUDE" -p "/ralph-iterate $EPIC" \
        --dangerously-skip-permissions \
        --output-format stream-json 2>&1 | \
        tee -a "$LOGFILE" | \
        jq -r --unbuffered '
            if .type == "assistant" and .message.content then
                .message.content[] |
                if .type == "text" then "💬 " + .text
                elif .type == "tool_use" then "🔧 " + .name + ": " + (.input | tostring | .[0:100])
                else empty
                end
            elif .type == "result" then
                "✅ Iteration complete"
            else empty
            end
        ' 2>/dev/null || true

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
