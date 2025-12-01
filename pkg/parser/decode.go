package parser

import (
	"encoding/binary"
	"fmt"
)

// DecodeInstructions decodes raw eBPF bytecode into Instruction objects.
// Handles standard 8-byte instructions and LD_IMM64 (16-byte) instructions.
func DecodeInstructions(raw []byte) ([]Instruction, error) {
	if len(raw)%8 != 0 {
		return nil, fmt.Errorf("bytecode length must be multiple of 8, got %d", len(raw))
	}

	var instructions []Instruction
	for i := 0; i < len(raw); {
		if len(raw)-i < 8 {
			return nil, fmt.Errorf("truncated instruction at offset %d", i)
		}

		opcode := raw[i]

		// Detect LD_IMM64: opcode 0x18 (BPF_LD | BPF_DW) is commonly used for 64-bit immediates.
		// If present, the instruction occupies 16 bytes (two 8-byte slots).
		if opcode == 0x18 {
			if len(raw)-i < 16 {
				return nil, fmt.Errorf("truncated LD_IMM64 at offset %d", i)
			}

			// Decode first 8-byte part using existing decoder
			insn, err := DecodeInstruction(raw[i : i+8])
			if err != nil {
				return nil, fmt.Errorf("failed to decode LD_IMM64 head at %d: %w", i, err)
			}

			// Combine lower and upper 32-bit immediates into a 64-bit immediate
			low := uint32(binary.LittleEndian.Uint32(raw[i+4 : i+8]))
			high := uint32(binary.LittleEndian.Uint32(raw[i+12 : i+16]))
			imm64 := int64(uint64(low) | (uint64(high) << 32))
			insn.Imm64 = imm64

			// Extend Raw to include the second 8 bytes
			insn.Raw = append(insn.Raw, raw[i+8:i+16]...)

			instructions = append(instructions, insn)
			i += 16
			continue
		}

		// Normal 8-byte instruction
		insn, err := DecodeInstruction(raw[i : i+8])
		if err != nil {
			return nil, fmt.Errorf("failed to decode instruction at offset %d: %w", i, err)
		}
		instructions = append(instructions, insn)
		i += 8
	}
	return instructions, nil
}
