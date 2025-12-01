#!/bin/bash
set -e

# --- Configuration ---

# Correct location of built binary:
BINARY_PATH="${BPFVA_PATH:-$(pwd)/bin/bpfva}"

TEST_DIR="test/compiled"
RESULTS_FILE="validation_results.txt"
BPF_TEST_FS="/sys/fs/bpf/bpfvatest"

echo "Running BPF Verifier Validation Test"
echo "===================================="
echo "NOTE: This requires 'sudo' access for 'bpftool' to load programs to the kernel."
echo "" > $RESULTS_FILE

# Dependency checks
command -v bpftool >/dev/null || { echo "bpftool required"; exit 1; }
command -v jq >/dev/null || { echo "jq required"; exit 1; }
test -f "$BINARY_PATH" || { echo "bpfva binary not found at $BINARY_PATH"; exit 1; }

sudo mkdir -p $(dirname $BPF_TEST_FS) 2>/dev/null || true

CORRECT=0
INCORRECT=0
UNCERTAIN=0
TOTAL=0

echo "Validation Test Results (Kernel)" >> $RESULTS_FILE
echo "------------------------------" >> $RESULTS_FILE

for prog in $TEST_DIR/*.o; do
    BASENAME=$(basename $prog)
    TOTAL=$((TOTAL + 1))

    # --- 1. RUN TOOL ---
    TOOL_OUTPUT=$($BINARY_PATH analyze "$prog" --json 2>/dev/null || echo "{}")

    PRED=$(echo "$TOOL_OUTPUT" | jq -r '.Prediction // "ERROR"')
    SCORE=$(echo "$TOOL_OUTPUT" | jq -r '.TotalScore // "N/A"')

    # --- 2. RUN KERNEL ---
    ACTUAL="FAIL"
    if sudo bpftool prog load "$prog" $BPF_TEST_FS 2>/dev/null; then
        ACTUAL="PASS"
        sudo rm -f $BPF_TEST_FS
    fi

    # --- 3. CLASSIFY ---
    STATUS="✗ INCORRECT"

    if [[ "$PRED" == "LIKELY_PASS" && "$ACTUAL" == "PASS" ]]; then
        STATUS="✓ CORRECT (True Positive)"
        CORRECT=$((CORRECT + 1))

    elif [[ "$PRED" == "WILL_FAIL" && "$ACTUAL" == "FAIL" ]]; then
        STATUS="✓ CORRECT (True Negative)"
        CORRECT=$((CORRECT + 1))

    elif [[ "$PRED" == "LIKELY_FAIL" && "$ACTUAL" == "FAIL" ]]; then
        STATUS="✓ CORRECT (True Negative)"
        CORRECT=$((CORRECT + 1))

    elif [[ "$PRED" == "MAY_PASS" ]]; then
        STATUS="○ UNCERTAIN"
        UNCERTAIN=$((UNCERTAIN + 1))
    else
        INCORRECT=$((INCORRECT + 1))
    fi

    # --- 4. OUTPUT ---
    {
        echo "$BASENAME"
        echo "  Tool Prediction: $PRED (Score: $SCORE)"
        echo "  Actual Outcome:  $ACTUAL"
        echo "  Result: $STATUS"
        echo ""
    } >> $RESULTS_FILE
done

TOTAL_PREDICTED=$((TOTAL - UNCERTAIN))
if [[ $TOTAL_PREDICTED -gt 0 ]]; then
    ACCURACY=$(echo "scale=2; $CORRECT * 100 / $TOTAL_PREDICTED" | bc)
else
    ACCURACY="N/A"
fi

{
    echo "--- Summary ---"
    echo "Total Programs: $TOTAL"
    echo "Correct Predictions (Excl. MAY_PASS): $CORRECT / $TOTAL_PREDICTED ($ACCURACY%)"
    echo "Incorrect Predictions: $INCORRECT"
    echo "Uncertain (MAY_PASS): $UNCERTAIN"
} >> $RESULTS_FILE

cat $RESULTS_FILE
