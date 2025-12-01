package verify

import (
	"github.com/Nash0810/BPF-Insight/pkg/cfg"
	"github.com/Nash0810/BPF-Insight/pkg/parser"
)

// Aliases for clarity
type Block = cfg.BasicBlock
type Instruction = parser.Instruction

// Rule definition
type Rule struct {
	Name         string
	Description  string
	Enabled      bool
	BlockCheck   func(block Block, ins Instruction) []string
	ProgramCheck func(blocks []Block) []string
}

// Global registry
var Rules = map[string]*Rule{}

// Register rules (called from init functions)
func RegisterRule(name string, rule *Rule) {
	Rules[name] = rule
}
