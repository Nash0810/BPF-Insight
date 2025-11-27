package parser

import (
	"github.com/cilium/ebpf/asm"
)

// ConvertToASM converts a slice of parser.Instruction to asm.Instruction
func ConvertToASM(insns []Instruction) []asm.Instruction {
	result := make([]asm.Instruction, len(insns))
	for i, ins := range insns {
		result[i] = asm.Instruction{
			OpCode: asm.OpCode(ins.Opcode),
			Dst:    asm.Register(ins.DstReg),
			Src:    asm.Register(ins.SrcReg),
			Offset: ins.OffsetVal,
			Constant: int64(ins.Imm),
		}
	}
	return result
}

