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

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
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

for script in "$SCRIPT_DIR"/*.sh; do
    [ -f "$script" ] || continue
    script_name=$(basename "$script")
    
    # Skip this validation script
    [ "$script_name" = "validate.sh" ] && continue
    
    # Find read commands that don't use /dev/tty
    # Exclude: comments, IFS= read for file processing, read -r line patterns for file reading
    bad_reads=$(grep -n "^\s*read " "$script" 2>/dev/null | grep -v "/dev/tty" | grep -v "^#" || true)
    
    if [ -n "$bad_reads" ]; then
        echo -e "${RED}✗ $script_name has read commands not using /dev/tty:${NC}"
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

for script in "$SCRIPT_DIR"/*.sh; do
    [ -f "$script" ] || continue
    script_name=$(basename "$script")
    
    if bash -n "$script" 2>&1; then
        echo -e "${GREEN}✓ $script_name${NC}"
    else
        echo -e "${RED}✗ $script_name has syntax errors${NC}"
        ((ERRORS++)) || true
    fi
done

echo ""

#------------------------------------------------------------------------------
# Check 3: Verify ask/ask_yes_no/ask_choice use stderr for prompts
#------------------------------------------------------------------------------
echo "Checking that prompt functions write to stderr..."

for script in "$SCRIPT_DIR/install.sh" "$SCRIPT_DIR/setup.sh"; do
    [ -f "$script" ] || continue
    script_name=$(basename "$script")

    # Check ask() function writes prompts to stderr (look for >&2 in the function body)
    # Use sed to extract function body and check for stderr redirect
    if sed -n '/^ask() {/,/^}/p' "$script" | grep -q ">&2"; then
        echo -e "${GREEN}✓ $script_name: ask() writes prompts to stderr${NC}"
    else
        echo -e "${RED}✗ $script_name: ask() should write prompts to stderr${NC}"
        ((ERRORS++)) || true
    fi

    # Check ask_yes_no() function writes prompts to stderr
    if sed -n '/^ask_yes_no() {/,/^}/p' "$script" | grep -q ">&2"; then
        echo -e "${GREEN}✓ $script_name: ask_yes_no() writes prompts to stderr${NC}"
    else
        echo -e "${RED}✗ $script_name: ask_yes_no() should write prompts to stderr${NC}"
        ((ERRORS++)) || true
    fi
done

# Check ask_choice in setup.sh
if sed -n '/^ask_choice() {/,/^}/p' "$SCRIPT_DIR/setup.sh" | grep -q ">&2"; then
    echo -e "${GREEN}✓ setup.sh: ask_choice() writes prompts to stderr${NC}"
else
    echo -e "${RED}✗ setup.sh: ask_choice() should write prompts to stderr${NC}"
    ((ERRORS++)) || true
fi

echo ""

#------------------------------------------------------------------------------
# Check 4: Functions that return values via stdout
#------------------------------------------------------------------------------
echo "Checking value-returning functions for stdout pollution..."

# Check select_project outputs prompts to stderr
if sed -n '/^select_project() {/,/^}/p' "$SCRIPT_DIR/install.sh" | grep -q ">&2"; then
    echo -e "${GREEN}✓ install.sh: select_project() writes prompts to stderr${NC}"
else
    echo -e "${RED}✗ install.sh: select_project() should write prompts to stderr${NC}"
    ((ERRORS++)) || true
fi

# Check install_or_update_ralph_loop outputs to stderr
if sed -n '/^install_or_update_ralph_loop() {/,/^}/p' "$SCRIPT_DIR/install.sh" | grep -q ">&2"; then
    echo -e "${GREEN}✓ install.sh: install_or_update_ralph_loop() writes output to stderr${NC}"
else
    echo -e "${RED}✗ install.sh: install_or_update_ralph_loop() should write output to stderr${NC}"
    ((ERRORS++)) || true
fi

echo ""

#------------------------------------------------------------------------------
# Check 5: git commands in value-returning functions should suppress output
#------------------------------------------------------------------------------
echo "Checking git commands for proper output redirection..."

# Check git pull commands in install_or_update_ralph_loop
# These should have >/dev/null 2>&1 since the function returns via stdout
func_content=$(sed -n '/^install_or_update_ralph_loop() {/,/^}/p' "$SCRIPT_DIR/install.sh")
bad_git_in_func=$(echo "$func_content" | grep "git pull" | grep -v ">/dev/null 2>&1" || true)

if [ -n "$bad_git_in_func" ]; then
    echo -e "${RED}✗ install_or_update_ralph_loop has git pull without full output suppression${NC}"
    echo "    Found: $bad_git_in_func"
    ((ERRORS++)) || true
else
    echo -e "${GREEN}✓ git pull commands properly suppress output in value-returning functions${NC}"
fi

echo ""

#------------------------------------------------------------------------------
# Check 6: echo statements with color codes should use -e flag
#------------------------------------------------------------------------------
echo "Checking echo statements with color codes use -e flag..."

color_check_errors=0
for script in "$SCRIPT_DIR"/*.sh; do
    [ -f "$script" ] || continue
    script_name=$(basename "$script")

    # Skip this validation script
    [ "$script_name" = "validate.sh" ] && continue

    # Find echo statements with color code variables (BOLD, NC, RED, GREEN, YELLOW, CYAN) that don't use -e
    # Pattern: echo followed by a quote (not -e or -en), containing ${BOLD}, ${NC}, ${RED}, ${GREEN}, ${YELLOW}, ${CYAN}
    bad_echo=$(grep -n 'echo "[^"]*\${\(BOLD\|NC\|RED\|GREEN\|YELLOW\|CYAN\)}' "$script" 2>/dev/null | grep -v 'echo -e' | grep -v 'echo -en' || true)

    if [ -n "$bad_echo" ]; then
        echo -e "${RED}✗ $script_name has echo with color codes missing -e flag:${NC}"
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

