package parser

import (
	"github.com/cilium/ebpf/asm"
)

// DecodeInstructions decodes raw eBPF bytecode into asm.Instruction objects.
// This uses cilium/ebpf's official BPF decoder, matching kernel semantics.
func DecodeInstructions(raw []byte) ([]asm.Instruction, error) {
	return asm.Parse(raw, asm.LE)
}
