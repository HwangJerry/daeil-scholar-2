#!/bin/bash
# Validates source files for size (150-500 lines) and header comments.
# Runs as a PostToolUse hook on Edit|Write operations.

INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // ""')

# Skip non-source files (only check .ts/.tsx in src/)
if [[ ! "$FILE_PATH" =~ \.tsx?$ ]] || [[ ! "$FILE_PATH" =~ /src/ ]]; then
  exit 0
fi

# Skip if file doesn't exist (deleted)
if [ ! -f "$FILE_PATH" ]; then
  exit 0
fi

WARNINGS=""
LINE_COUNT=$(wc -l < "$FILE_PATH" | tr -d ' ')

# Size check (SHOULD: 150-500 lines)
if [ "$LINE_COUNT" -gt 500 ]; then
  WARNINGS+="⚠ SIZE VIOLATION: ${FILE_PATH##*/} has $LINE_COUNT lines (max recommended: 500). [FIX REQUIRED] Split this file into smaller modules following the project's domain-based directory structure. Do not ask for confirmation — fix inline and continue.\n"
elif [ "$LINE_COUNT" -lt 150 ] && [ "$LINE_COUNT" -gt 30 ]; then
  WARNINGS+="ℹ SIZE NOTE: ${FILE_PATH##*/} has $LINE_COUNT lines (recommended: 150-500). [INFO ONLY] No action needed. Do not merge this file.\n"
fi

# Header comment check (MUST: first 10 lines should contain a comment block)
HEAD=$(head -10 "$FILE_PATH")
if ! echo "$HEAD" | grep -qE '^\s*(//|/\*|\*)'; then
  WARNINGS+="⚠ HEADER MISSING: ${FILE_PATH##*/} lacks a header comment. [FIX REQUIRED] Add a header comment to line 1 in the format: // FileName — brief purpose description. Do not ask for confirmation — fix inline and continue.\n"
fi

if [ -n "$WARNINGS" ]; then
  FOOTER="Auto-fix all [FIX REQUIRED] items immediately without asking the user. Continue with the current task after fixing."
  CONTEXT=$(printf "$WARNINGS")
  echo "{\"hookSpecificOutput\":{\"hookEventName\":\"PostToolUse\",\"additionalContext\":\"[Code Compliance Check]\\n${CONTEXT}\\n${FOOTER}\"}}"
fi

exit 0
