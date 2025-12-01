#!/usr/bin/env bash
set -euo pipefail
TEST_DIR="test/compiled"
BPF_TEST_FS="/sys/fs/bpf/bpfvatest"

for prog in "$TEST_DIR"/*.o; do
  name=$(basename "$prog")
  echo "--- $name ---"
  if sudo bpftool prog load "$prog" $BPF_TEST_FS 2> /tmp/bpf_err.log; then
    echo "Loaded OK (unexpected)"
    sudo rm -f $BPF_TEST_FS
  else
    echo "Load failed, verifier output:" 
    sed -n '1,200p' /tmp/bpf_err.log
  fi
  echo ""
done
