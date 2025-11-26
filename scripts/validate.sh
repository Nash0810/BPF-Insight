#!/bin/bash
# File: scripts/validate.sh
# Purpose: Compares bpfva predictions against the actual kernel verifier (bpftool).
#
# Requirements:
# 1. bpftool must be installed and executable (requires root/sudo).
# 2. jq must be installed for JSON parsing.
# 3. Test programs must be compiled (e.g., using 'make compile-tests').

# Exit immediately if a command exits with a non-zero status.
set -e

# --- Configuration ---
mv ./bin/bpfva .

# Change the script's path variable
BINARY_PATH="${BPFVA_PATH:-./bin/bpfva}"
TEST_DIR="test/compiled"
RESULTS_FILE="validation_results.txt"
BPF_TEST_FS="/sys/fs/bpf/bpfvatest"

echo "Running BPF Verifier Validation Test"
echo "===================================="
echo "NOTE: This requires 'sudo' access for 'bpftool' to load programs to the kernel."
echo "" > $RESULTS_FILE # Clear results file

# Check dependencies
if ! command -v $BINARY_PATH &> /dev/null
then
    echo "Error: bpfva binary not found at $BINARY_PATH. Please run 'make build'."
    exit 1
fi
if ! command -v bpftool &> /dev/null
then
    echo "Error: bpftool command not found. Please install it (requires Linux kernel source and compilation)."
    exit 1
fi
if ! command -v jq &> /dev/null
then
    echo "Error: jq command not found. Please install it (e.g., 'sudo apt install jq')."
    exit 1
fi

# Attempt to create BPF filesystem mount point (if it doesn't exist)
sudo mkdir -p $(dirname $BPF_TEST_FS) 2>/dev/null || true

CORRECT=0
INCORRECT=0
UNCERTAIN=0
TOTAL=0

echo "Validation Test Results (Kernel)" >> $RESULTS_FILE
echo "------------------------------" >> $RESULTS_FILE

# Main loop to process all compiled test programs
for prog in $TEST_DIR/*.o; do
    BASENAME=$(basename $prog)
    TOTAL=$((TOTAL + 1))
    
    # 1. Get tool prediction
    TOOL_OUTPUT=$($BINARY_PATH analyze $prog -f json || echo "{}")
    PRED=$(echo "$TOOL_OUTPUT" | jq -r '.prediction' 2>/dev/null)
    SCORE=$(echo "$TOOL_OUTPUT" | jq -r '.score' 2>/dev/null)
    
    if [ "$PRED" == "null" ] || [ -z "$PRED" ]; then
        PRED="ERROR"
        SCORE="N/A"
    fi

    # 2. Try to load with bpftool (Actual Outcome)
    ACTUAL="FAIL"
    # Suppress bpftool's verifier error messages in the console
    if sudo bpftool prog load $prog $BPF_TEST_FS 2>/dev/null; then
        ACTUAL="PASS"
        # Clean up the loaded program immediately
        sudo rm $BPF_TEST_FS
    fi
    
    # 3. Compare and categorize result (Section 5.12, Example Output)
    STATUS="✗ INCORRECT"
    
    # Check for True Positives/Negatives
    if [ "$PRED" == "LIKELY_PASS" ] && [ "$ACTUAL" == "PASS" ]; then
        STATUS="✓ CORRECT (True Positive)"
        CORRECT=$((CORRECT + 1))
    elif [ "$PRED" == "WILL_FAIL" ] && [ "$ACTUAL" == "FAIL" ]; then
        STATUS="✓ CORRECT (True Negative)"
        CORRECT=$((CORRECT + 1))
    elif [ "$PRED" == "LIKELY_FAIL" ] && [ "$ACTUAL" == "FAIL" ]; then
        STATUS="✓ CORRECT (True Negative)"
        CORRECT=$((CORRECT + 1))
    elif [ "$PRED" == "MAY_PASS" ]; then
        STATUS="○ UNCERTAIN (Acceptable)"
        UNCERTAIN=$((UNCERTAIN + 1))
    fi
    
    if [[ "$STATUS" == "✗ INCORRECT" ]]; then
        INCORRECT=$((INCORRECT + 1))
    fi

    echo "$BASENAME" >> $RESULTS_FILE
    echo "  Tool Prediction: $PRED (Score: $SCORE)" >> $RESULTS_FILE
    echo "  Actual Outcome:  $ACTUAL" >> $RESULTS_FILE
    echo "  Result: $STATUS" >> $RESULTS_FILE
    echo "" >> $RESULTS_FILE
done

# Calculate accuracy summary
TOTAL_PREDICTED=$((TOTAL - UNCERTAIN))
ACCURACY=$(echo "scale=2; $CORRECT * 100 / $TOTAL_PREDICTED" | bc -l 2>/dev/null || echo "N/A")

echo "--- Summary ---" >> $RESULTS_FILE
echo "Total Programs: $TOTAL" >> $RESULTS_FILE
echo "Correct Predictions (Excl. MAY_PASS): $CORRECT / $TOTAL_PREDICTED ($ACCURACY%)" >> $RESULTS_FILE
echo "Incorrect Predictions: $INCORRECT" >> $RESULTS_FILE
echo "Uncertain (MAY_PASS): $UNCERTAIN" >> $RESULTS_FILE

cat $RESULTS_FILE