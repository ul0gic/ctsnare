#!/bin/bash
# Stop hook: Verify work before Claude stops
# Customize the build/test commands below for your project
# Exit 2 + stderr to block Claude from stopping and force it to fix issues

# ============================================================
# CONFIGURE THESE FOR YOUR PROJECT
# Uncomment and edit the commands that apply
# ============================================================

export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH
BUILD_CMD="go build -o ctsnare ./cmd/ctsnare"
TEST_CMD="go test ./..."
LINT_CMD="go vet ./... && golangci-lint run ./..."

# ============================================================
# VERIFICATION LOGIC (no changes needed below)
# ============================================================

ERRORS=""

if [ -n "$BUILD_CMD" ]; then
  OUTPUT=$(eval "$BUILD_CMD" 2>&1)
  if [ $? -ne 0 ]; then
    ERRORS="${ERRORS}Build failed:\n${OUTPUT}\n\n"
  fi
fi

if [ -n "$TEST_CMD" ]; then
  OUTPUT=$(eval "$TEST_CMD" 2>&1)
  if [ $? -ne 0 ]; then
    ERRORS="${ERRORS}Tests failed:\n${OUTPUT}\n\n"
  fi
fi

if [ -n "$LINT_CMD" ]; then
  OUTPUT=$(eval "$LINT_CMD" 2>&1)
  if [ $? -ne 0 ]; then
    ERRORS="${ERRORS}Lint errors:\n${OUTPUT}\n\n"
  fi
fi

if [ -n "$ERRORS" ]; then
  echo -e "$ERRORS" >&2
  exit 2  # Blocks Claude from stopping — forces it to fix
fi

exit 0
