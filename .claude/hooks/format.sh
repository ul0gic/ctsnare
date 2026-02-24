#!/bin/bash
# PostToolUse hook: Auto-format files after Edit/Write
# Receives JSON on stdin with tool_input.file_path
# Customize the formatter commands below for your project

FILE_PATH=$(jq -r '.tool_input.file_path // empty')

if [ -z "$FILE_PATH" ] || [ ! -f "$FILE_PATH" ]; then
  exit 0
fi

EXT="${FILE_PATH##*.}"

case "$EXT" in
  ts|tsx|js|jsx|json|css|scss|html|md)
    if command -v prettier &>/dev/null; then
      prettier --write "$FILE_PATH" 2>/dev/null
    elif command -v npx &>/dev/null; then
      npx prettier --write "$FILE_PATH" 2>/dev/null
    fi
    ;;
  rs)
    if command -v rustfmt &>/dev/null; then
      rustfmt "$FILE_PATH" 2>/dev/null
    fi
    ;;
  go)
    if command -v gofmt &>/dev/null; then
      gofmt -w "$FILE_PATH" 2>/dev/null
    fi
    ;;
  swift)
    if command -v swift-format &>/dev/null; then
      swift-format format -i "$FILE_PATH" 2>/dev/null
    fi
    ;;
  py)
    if command -v ruff &>/dev/null; then
      ruff format --quiet "$FILE_PATH" 2>/dev/null
    elif command -v black &>/dev/null; then
      black --quiet "$FILE_PATH" 2>/dev/null
    fi
    ;;
esac

exit 0
