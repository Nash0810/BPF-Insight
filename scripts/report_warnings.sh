#!/bin/bash
set -e

BINARY_PATH="${BPFVA_PATH:-$(pwd)/bin/bpfva}"
TEST_DIR="test/compiled"
OUT_DIR="/tmp/verify_outputs"
SUMMARY="/tmp/verify_summary.txt"

mkdir -p "$OUT_DIR"
> "$SUMMARY"

for prog in $TEST_DIR/*.o; do
    base=$(basename "$prog")
    echo "Processing $base"
    # Run verify in JSON mode
    $BINARY_PATH verify "$prog" --json > "$OUT_DIR/$base.json" 2>/dev/null || true

    # Extract top warnings
    echo "Program: $base" >> "$SUMMARY"
    jq -r '.BlockWarnings[]?.Message' "$OUT_DIR/$base.json" | sed 's/^/  Block: /' >> "$SUMMARY" || true
    jq -r '.ProgramWarnings[]?' "$OUT_DIR/$base.json" | sed 's/^/  Program: /' >> "$SUMMARY" || true
    echo "" >> "$SUMMARY"

done

cat "$SUMMARY"
