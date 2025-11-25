package verify

import "github.com/Nash0810/BPF-Insight/pkg/parser"

type RegClass int

const (
	RegUnknown RegClass = iota
	RegScalar
	RegPtr
)

type RegState struct {
	Regs [11]RegClass
}

// Initialize ABI: R1 = ctx pointer, R10 = frame pointer
func NewRegState() *RegState {
	s := &RegState{}
	s.Regs[1] = RegPtr
	s.Regs[10] = RegPtr
	return s
}

func (s *RegState) Apply(ins parser.Instruction) {
	dst := ins.Dst()
	src := ins.Src()

	if ins.IsMove() {
		s.Regs[dst] = s.Regs[src]
		return
	}

	if ins.IsALU() {
		// ALU does not change pointer->scalar or scalar->pointer
		return
	}

	if ins.IsHelperCall() {
		if ins.Imm == 1 {
			s.Regs[0] = RegPtr // map_lookup returns pointer
		} else {
			s.Regs[0] = RegScalar
		}
	}
}
