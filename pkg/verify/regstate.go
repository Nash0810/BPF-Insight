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

// NewUnknownState returns a RegState with unknown classes except ABI-known regs
func NewUnknownState() *RegState {
	s := &RegState{}
	s.Regs[1] = RegPtr
	s.Regs[10] = RegPtr
	return s
}

// Clone returns a copy of the RegState
func (s *RegState) Clone() *RegState {
	if s == nil {
		return nil
	}
	c := &RegState{}
	copy(c.Regs[:], s.Regs[:])
	return c
}

// Merge merges another RegState into this one using a conservative lattice:
// if values equal, keep; otherwise set to RegUnknown. Returns true if changed.
func (s *RegState) Merge(other *RegState) bool {
	changed := false
	if other == nil {
		return false
	}
	for i := 0; i < len(s.Regs); i++ {
		a := s.Regs[i]
		b := other.Regs[i]
		if a == b {
			continue
		}
		// If one is zero (unset) use the other
		if a == RegUnknown {
			if b != RegUnknown {
				s.Regs[i] = b
				changed = true
			}
			continue
		}
		if b == RegUnknown {
			continue
		}
		// Different non-unknown values -> set to unknown
		if a != b {
			s.Regs[i] = RegUnknown
			changed = true
		}
	}
	return changed
}

// Equals checks equality
func (s *RegState) Equals(other *RegState) bool {
	if s == nil || other == nil {
		return s == other
	}
	for i := 0; i < len(s.Regs); i++ {
		if s.Regs[i] != other.Regs[i] {
			return false
		}
	}
	return true
}

func (s *RegState) Apply(ins parser.Instruction) {
	dst := ins.Dst()
	src := ins.Src()

	if ins.IsMove() {
		s.Regs[dst] = s.Regs[src]
		return
	}

	if ins.IsALU() {
		// Only treat ALU as pointer-modifying when the operation is one known to
		// modify pointer values (add/sub/and/or/xor/lsh/rsh).
		if ins.IsPointerModifyingALU() {
			if src >= 0 && src < len(s.Regs) && s.Regs[src] == RegPtr {
				s.Regs[dst] = RegUnknown
				return
			}
			if dst >= 0 && dst < len(s.Regs) && s.Regs[dst] == RegPtr {
				s.Regs[dst] = RegUnknown
				return
			}
			// If neither operand is known pointer, result is scalar
			s.Regs[dst] = RegScalar
			return
		}

		// Non-pointer-modifying ALU ops: treat result as scalar
		s.Regs[dst] = RegScalar
		return
	}

	if ins.IsHelperCall() {
		if ins.Imm == 1 {
			s.Regs[0] = RegPtr // map_lookup returns pointer
		} else {
			s.Regs[0] = RegScalar
		}
	}

	// Loads write into dst — treat as scalar by default
	if ins.IsLoad() {
		// Conservative: treat loads as scalar by default. Inferring pointer from a load
		// is error-prone without BTF or additional context. Map lookups explicitly set R0.
		s.Regs[dst] = RegScalar
	}
}
