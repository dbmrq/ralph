#!/bin/bash
#==============================================================================
# Ralph Loop - Script Validation
#==============================================================================
# This script checks for common issues that can break piped execution:
# 1. read commands not using /dev/tty
# 2. Functions that return values via stdout but have prompts going to stdout
# 3. git commands outputting to stdout in value-returning functions
#==============================================================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Determine repo root directory (one level up from tests/)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CORE_DIR="$REPO_ROOT/core"
ERRORS=0
WARNINGS=0

echo "═══════════════════════════════════════════════════════════════"
echo "   Ralph Loop Script Validation"
echo "═══════════════════════════════════════════════════════════════"
echo ""

#------------------------------------------------------------------------------
# Check 1: All read commands should use /dev/tty (except in specific contexts)
#------------------------------------------------------------------------------
echo "Checking for read commands not using /dev/tty..."

# Check install.sh
if [ -f "$REPO_ROOT/install.sh" ]; then
    bad_reads=$(grep -n "^\s*read " "$REPO_ROOT/install.sh" 2>/dev/null | grep -v "/dev/tty" | grep -v "^#" || true)
    if [ -n "$bad_reads" ]; then
        echo -e "${RED}✗ install.sh has read commands not using /dev/tty:${NC}"
        echo "$bad_reads" | while read line; do
            echo "    $line"
        done
        ((ERRORS++)) || true
    fi
fi

# Check lib/*.sh
for script in "$REPO_ROOT/lib"/*.sh; do
    [ -f "$script" ] || continue
    script_name=$(basename "$script")

    bad_reads=$(grep -n "^\s*read " "$script" 2>/dev/null | grep -v "/dev/tty" | grep -v "^#" || true)

    if [ -n "$bad_reads" ]; then
        echo -e "${RED}✗ lib/$script_name has read commands not using /dev/tty:${NC}"
        echo "$bad_reads" | while read line; do
            echo "    $line"
        done
        ((ERRORS++)) || true
    fi
done

echo ""

#------------------------------------------------------------------------------
# Check 2: Syntax validation
#------------------------------------------------------------------------------
echo "Checking bash syntax..."

# Check install.sh
if [ -f "$REPO_ROOT/install.sh" ]; then
    if bash -n "$REPO_ROOT/install.sh" 2>&1; then
        echo -e "${GREEN}✓ install.sh${NC}"
    else
        echo -e "${RED}✗ install.sh has syntax errors${NC}"
        ((ERRORS++)) || true
    fi
fi

# Check core/*.sh
for script in "$CORE_DIR"/*.sh; do
    [ -f "$script" ] || continue
    script_name=$(basename "$script")

    if bash -n "$script" 2>&1; then
        echo -e "${GREEN}✓ core/$script_name${NC}"
    else
        echo -e "${RED}✗ core/$script_name has syntax errors${NC}"
        ((ERRORS++)) || true
    fi
done

# Check lib/*.sh
for script in "$REPO_ROOT/lib"/*.sh; do
    [ -f "$script" ] || continue
    script_name=$(basename "$script")

    if bash -n "$script" 2>&1; then
        echo -e "${GREEN}✓ lib/$script_name${NC}"
    else
        echo -e "${RED}✗ lib/$script_name has syntax errors${NC}"
        ((ERRORS++)) || true
    fi
done

echo ""

#------------------------------------------------------------------------------
# Check 3: Verify ask/ask_yes_no/ask_choice use stderr for prompts in lib/common.sh
#------------------------------------------------------------------------------
echo "Checking that prompt functions write to stderr..."

if [ -f "$REPO_ROOT/lib/common.sh" ]; then
    # Check ask() function writes prompts to stderr
    if sed -n '/^ask() {/,/^}/p' "$REPO_ROOT/lib/common.sh" | grep -q ">&2"; then
        echo -e "${GREEN}✓ lib/common.sh: ask() writes prompts to stderr${NC}"
    else
        echo -e "${RED}✗ lib/common.sh: ask() should write prompts to stderr${NC}"
        ((ERRORS++)) || true
    fi

    # Check ask_yes_no() function writes prompts to stderr
    if sed -n '/^ask_yes_no() {/,/^}/p' "$REPO_ROOT/lib/common.sh" | grep -q ">&2"; then
        echo -e "${GREEN}✓ lib/common.sh: ask_yes_no() writes prompts to stderr${NC}"
    else
        echo -e "${RED}✗ lib/common.sh: ask_yes_no() should write prompts to stderr${NC}"
        ((ERRORS++)) || true
    fi

    # Check ask_choice() function writes prompts to stderr
    if sed -n '/^ask_choice() {/,/^}/p' "$REPO_ROOT/lib/common.sh" | grep -q ">&2"; then
        echo -e "${GREEN}✓ lib/common.sh: ask_choice() writes prompts to stderr${NC}"
    else
        echo -e "${RED}✗ lib/common.sh: ask_choice() should write prompts to stderr${NC}"
        ((ERRORS++)) || true
    fi
fi

echo ""

#------------------------------------------------------------------------------
# Check 4: echo statements with color codes should use -e flag
#------------------------------------------------------------------------------
echo "Checking echo statements with color codes use -e flag..."

color_check_errors=0

# Check install.sh
if [ -f "$REPO_ROOT/install.sh" ]; then
    bad_echo=$(grep -n 'echo "[^"]*\${\(BOLD\|NC\|RED\|GREEN\|YELLOW\|CYAN\)}' "$REPO_ROOT/install.sh" 2>/dev/null | grep -v 'echo -e' | grep -v 'echo -en' || true)
    if [ -n "$bad_echo" ]; then
        echo -e "${RED}✗ install.sh has echo with color codes missing -e flag:${NC}"
        echo "$bad_echo" | while read line; do
            echo "    $line"
        done
        ((color_check_errors++)) || true
        ((ERRORS++)) || true
    fi
fi

# Check lib/*.sh
for script in "$REPO_ROOT/lib"/*.sh; do
    [ -f "$script" ] || continue
    script_name=$(basename "$script")

    bad_echo=$(grep -n 'echo "[^"]*\${\(BOLD\|NC\|RED\|GREEN\|YELLOW\|CYAN\)}' "$script" 2>/dev/null | grep -v 'echo -e' | grep -v 'echo -en' || true)

    if [ -n "$bad_echo" ]; then
        echo -e "${RED}✗ lib/$script_name has echo with color codes missing -e flag:${NC}"
        echo "$bad_echo" | while read line; do
            echo "    $line"
        done
        ((color_check_errors++)) || true
        ((ERRORS++)) || true
    fi
done

# If no errors found, show success
if [ $color_check_errors -eq 0 ]; then
    echo -e "${GREEN}✓ All echo statements with color codes use -e flag${NC}"
fi

echo ""

#------------------------------------------------------------------------------
# Check 7: Agent availability (informational)
#------------------------------------------------------------------------------
echo "Checking available AI agents..."

cursor_available=false
auggie_available=false

if command -v agent &> /dev/null; then
    echo -e "${GREEN}✓ Cursor CLI (agent) is available${NC}"
    cursor_available=true
else
    echo -e "${YELLOW}○ Cursor CLI (agent) not found${NC}"
fi

if command -v auggie &> /dev/null; then
    echo -e "${GREEN}✓ Augment CLI (auggie) is available${NC}"
    auggie_available=true
else
    echo -e "${YELLOW}○ Augment CLI (auggie) not found${NC}"
fi

if [ "$cursor_available" = false ] && [ "$auggie_available" = false ]; then
    echo ""
    echo -e "${YELLOW}⚠ No AI agent CLI detected. You'll need to install one before running Ralph Loop.${NC}"
    ((WARNINGS++)) || true
fi

echo ""

#------------------------------------------------------------------------------
# Summary
#------------------------------------------------------------------------------
echo "═══════════════════════════════════════════════════════════════"
if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo -e "${GREEN}✓ All checks passed!${NC}"
    exit 0
elif [ $ERRORS -eq 0 ]; then
    echo -e "${YELLOW}⚠ $WARNINGS warning(s) found${NC}"
    exit 0
else
    echo -e "${RED}✗ $ERRORS error(s), $WARNINGS warning(s) found${NC}"
    exit 1
fi

