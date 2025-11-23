package parser

// Instruction represents a decoded eBPF instruction.
// Fields fully specified in project plan.
type Instruction struct {
    Offset    int
    Opcode    uint8
    DstReg    uint8
    SrcReg    uint8
    OffsetVal int16
    Imm       int32
    ImmHigh   int32 // for LD_IMM64
}

// IsJump reports whether the opcode is a jump instruction.
// TODO: Implement in Iteration 1
func (i *Instruction) IsJump() bool {
    return false
}

// IsExit reports EXIT instruction.
// TODO: Implement in Iteration 1
func (i *Instruction) IsExit() bool {
    return false
}

func (i *Instruction) String() string {
    return "<unimplemented instruction formatting>"
}
