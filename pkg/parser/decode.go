package parser

import (
	"fmt"
)

// raw is a flat byte slice containing all instructions
// We split it into 8-byte BPF instructions
func DecodeInstructions(raw []byte) ([]Instruction, error) {

	if len(raw)%8 != 0 {
		return nil, fmt.Errorf("instruction data is not aligned to 8 bytes")
	}

	insns := []Instruction{}

	for i := 0; i < len(raw); i += 8 {
		ins, err := DecodeInstruction(raw[i : i+8])
		if err != nil {
			return nil, fmt.Errorf("decode error at %d: %w", i/8, err)
		}
		insns = append(insns, ins)
	}

	return insns, nil
}
