#!/usr/bin/env bash
set -euo pipefail

BINARY_PATH="${BPFVA_PATH:-$(pwd)/bin/bpfva}"
TEST_DIR="test/compiled"
BPF_TEST_FS="/sys/fs/bpf/bpfvatest"

if ! command -v jq >/dev/null; then
  echo "jq required"
  exit 1
fi
if ! command -v bpftool >/dev/null; then
  echo "bpftool required"
  exit 1
fi

printf "Diagnosing predictions for programs in %s\n" "$TEST_DIR"
printf "Using binary: %s\n\n" "$BINARY_PATH"

for prog in "$TEST_DIR"/*.o; do
  name=$(basename "$prog")
  echo "--- $name ---"

  out=$($BINARY_PATH analyze --json "$prog" 2>/dev/null || echo "{}")
  # extract fields with defaults
  instr=$(echo "$out" | jq -r '.instruction_count // 0')
  maxdepth=$(echo "$out" | jq -r '.max_depth // 0')
  loops=$(echo "$out" | jq -r '.loops_detected // 0')
  avgbranch=$(echo "$out" | jq -r '.avg_branching // 0')
  helpers=$(echo "$out" | jq -r '.helper_call_count // 0')
  rulepen=$(echo "$out" | jq -r '.rule_penalty // 0')
  cfgscore=$(echo "$out" | jq -r '.cfg_score // 0')
  totalscore=$(echo "$out" | jq -r '.TotalScore // .score // 0')
  pred=$(echo "$out" | jq -r '.Prediction // .prediction // "ERROR"')

  # compute components locally (same formula as analyzer)
  instructionScore=$(awk -v i=$instr 'BEGIN{printf "%.3f", ((i/1000000.0*40.0)>40?40:(i/1000000.0*40.0)) }')
  depthScore=$(awk -v d=$maxdepth 'BEGIN{md=100; v=(d/md*15.0); if(v>15)v=15; printf "%.3f", v }')
  loopScore=$(awk -v l=$loops 'BEGIN{v=l*5.0; if(v>10) v=10; printf "%.3f", v }')
  avgBranchScore=$(awk -v a=$avgbranch 'BEGIN{v=a*5.0; if(v>10) v=10; printf "%.3f", v }')
  helperScore=$(awk -v h=$helpers 'BEGIN{v=(h/50.0*5.0); if(v>5) v=5; printf "%.3f", v }')
  cfgscore_local=$(awk -v i=$instructionScore -v d=$depthScore -v l=$loopScore -v b=$avgBranchScore -v h=$helperScore 'BEGIN{printf "%.3f", i+d+l+b+h}')
  totalscore_local=$(awk -v c=$cfgscore_local -v r=$rulepen 'BEGIN{v=c+r; if(v>100) v=100; printf "%.3f", v }')

  echo "Prediction: $pred (TotalScore reported: $totalscore)"
  # Determine actual outcome by attempting to load
  ACTUAL="FAIL"
  if sudo bpftool prog load "$prog" $BPF_TEST_FS 2>/dev/null; then
    ACTUAL="PASS"
    sudo rm -f $BPF_TEST_FS
  fi
  echo "Actual (bpftool): $ACTUAL"

  # Print component table
  printf "Components:\n"
  printf "  instruction_count: %d -> instructionScore: %s\n" "$instr" "$instructionScore"
  printf "  max_depth: %d -> depthScore: %s\n" "$maxdepth" "$depthScore"
  printf "  loops_detected: %d -> loopScore: %s\n" "$loops" "$loopScore"
  printf "  avg_branching: %s -> avgBranchScore: %s\n" "$avgbranch" "$avgBranchScore"
  printf "  helper_call_count: %d -> helperScore: %s\n" "$helpers" "$helperScore"
  printf "  cfg_score (reported): %s, cfg_score (calculated): %s\n" "$cfgscore" "$cfgscore_local"
  printf "  rule_penalty: %s\n" "$rulepen"
  printf "  TotalScore (reported): %s, TotalScore (calculated): %s\n" "$totalscore" "$totalscore_local"

  # Highlight differences if any
  diff_flag=0
  cmp=$(awk -v a=$totalscore -v b=$totalscore_local 'BEGIN{if((a-b)>0.01|| (b-a)>0.01) print 1; else print 0}')
  if [ "$cmp" -eq 1 ]; then
    echo "  NOTE: Reported TotalScore differs from locally recalculated value"
    diff_flag=1
  fi

  # If mispredicted, print a short diagnostic hint
  if [ "$pred" == "LIKELY_PASS" ] && [ "$ACTUAL" == "FAIL" ]; then
    echo "  -> MISPREDICTION: Tool thought PASS but kernel rejects. Check rule penalties and verifier warnings."
  elif [[ ("$pred" == "LIKELY_FAIL" || "$pred" == "WILL_FAIL") && "$ACTUAL" == "PASS" ]]; then
    echo "  -> MISPREDICTION: Tool thought FAIL but kernel accepted. Check thresholds and rule false positives."
  fi

  echo ""
done
