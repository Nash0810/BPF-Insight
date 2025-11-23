package parser

import (
	"encoding/binary"
	"fmt"
)

type Instruction struct {
	Opcode    uint8
	DstReg    uint8
	SrcReg    uint8
	OffsetVal int16
	Imm       int32
	Raw       []byte
}

// -------- Basic Getters --------

func (i Instruction) Dst() int {
	return int(i.DstReg)
}

func (i Instruction) Src() int {
	return int(i.SrcReg)
}

func (i Instruction) Off() int {
	return int(i.OffsetVal)
}

// Size for BPF LOAD/STORE instructions (encoded in opcode upper bits)
func (i Instruction) Size() int {
	// BPF size bits: opcode >> 3 & 0x3
	sz := (i.Opcode >> 3) & 0x03
	switch sz {
	case 0:
		return 1
	case 1:
		return 2
	case 2:
		return 4
	case 3:
		return 8
	}
	return 0
}

// -------- Instruction Class Helpers (BPF encoding) --------

// Instruction class (lowest 3 bits of opcode)
func (i Instruction) Class() uint8 {
	return i.Opcode & 0x07
}

// BPF classes
const (
	BPF_LD  = 0x00
	BPF_LDX = 0x01
	BPF_ST  = 0x02
	BPF_STX = 0x03
	BPF_ALU = 0x04
	BPF_JMP = 0x05
	BPF_RET = 0x06
	BPF_ALU64 = 0x07
)

// -------- Type Checkers --------

func (i Instruction) IsHelperCall() bool {
	return i.Class() == BPF_JMP && (i.Opcode&0xf0) == 0x80 // 0x85 = CALL
}

func (i Instruction) IsJump() bool {
	return i.Class() == BPF_JMP && !i.IsHelperCall() && !i.IsExit()
}

func (i Instruction) IsExit() bool {
	return i.Opcode == 0x95
}

func (i Instruction) IsMove() bool {
	// MOV opcode family = 0xb*
	return (i.Opcode & 0xf0) == 0xb0
}

func (i Instruction) IsALU() bool {
	// ALU or ALU64 classes
	class := i.Class()
	return class == BPF_ALU || class == BPF_ALU64
}

func (i Instruction) IsLoad() bool {
	class := i.Class()
	return class == BPF_LD || class == BPF_LDX
}

func (i Instruction) IsStore() bool {
	class := i.Class()
	return class == BPF_ST || class == BPF_STX
}

// -------- Decode ELF raw instruction --------

func DecodeInstruction(raw []byte) (Instruction, error) {
	if len(raw) != 8 {
		return Instruction{}, fmt.Errorf("invalid instruction size")
	}

	opcode := raw[0]
	dst := raw[1] & 0x0f
	src := (raw[1] >> 4) & 0x0f
	off := int16(binary.LittleEndian.Uint16(raw[2:4]))
	imm := int32(binary.LittleEndian.Uint32(raw[4:8]))

	return Instruction{
		Opcode:    opcode,
		DstReg:    dst,
		SrcReg:    src,
		OffsetVal: off,
		Imm:       imm,
		Raw:       raw,
	}, nil
}

func (i Instruction) IsPtrArithmetic() bool {
    class := i.Class()

    if class != BPF_ALU && class != BPF_ALU64 {
        return false
    }

    op := i.Opcode & 0xf0 // ALU op code

    switch op {
    case 0x00: // ADD
    case 0x10: // SUB
    case 0x20: // MUL
    case 0x30: // DIV
    case 0x40: // OR
    case 0x50: // AND
    case 0x60: // LSH (<<)
    case 0x70: // RSH (>>)
    case 0xa0: // XOR
    case 0xc0: // ARSH
        return true
    }

    return false
}
