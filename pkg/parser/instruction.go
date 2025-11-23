package parser

import (
	"encoding/binary"
	"fmt"
)

// eBPF instruction is 8 bytes:
//  0: opcode
//  1: registers (src in upper nibble, dst in lower nibble)
//  2-3: offset (int16)
//  4-7: immediate (int32)
//
// LD_IMM64 uses two instructions (16 bytes total)

type Instruction struct {
	Opcode    uint8
	Regs      uint8
	OffsetVal int16
	Imm       int32
	Offset    int // program counter (per instruction)
}

func DecodeInstructions(raw []byte) ([]Instruction, error) {
	var insns []Instruction

	for pc := 0; pc < len(raw); {
		if pc+8 > len(raw) {
			return nil, fmt.Errorf("truncated instruction at offset %d", pc)
		}

		ins := Instruction{
			Opcode:    raw[pc],
			Regs:      raw[pc+1],
			OffsetVal: int16(binary.LittleEndian.Uint16(raw[pc+2:])),
			Imm:       int32(binary.LittleEndian.Uint32(raw[pc+4:])),
			Offset:    pc / 8,
		}

		insns = append(insns, ins)
		pc += 8

		// Handle 16-byte LD_IMM64 (0x18)
		if ins.Opcode == 0x18 {
			if pc+8 > len(raw) {
				return nil, fmt.Errorf("truncated LD_IMM64 at offset %d", pc)
			}

			upper := binary.LittleEndian.Uint32(raw[pc+4:])
			fullImm := (uint64(uint32(ins.Imm))) | (uint64(upper) << 32)
			ins.Imm = int32(fullImm) // truncated but OK for CFG

			insns[len(insns)-1] = ins // update
			pc += 8
		}
	}

	return insns, nil
}

// Extract destination register from Regs
func (i Instruction) Dst() int {
	return int(i.Regs & 0x0f)
}

// Extract source register from Regs
func (i Instruction) Src() int {
	return int((i.Regs >> 4) & 0x0f)
}

func (i Instruction) IsHelperCall() bool {
	return i.Opcode == 0x85 // BPF_CALL
}

func (i Instruction) IsExit() bool {
	return i.Opcode == 0x95 // BPF_EXIT
}

// Correct jump detection (NO false positives on CALL)
func (i Instruction) IsJump() bool {
	switch i.Opcode {
	case 0x05: // JA (jump always)
		return true
	case 0x15, 0x25, 0x35, 0x45, 0x55: // JEQ, JGT, JGE, JSET, JNE
		return true
	}
	return false
}
