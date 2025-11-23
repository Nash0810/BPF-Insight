package parser

import (
	"errors"
)

// ELFParser loads eBPF bytecode from ELF files.
type ELFParser struct {
    FilePath string
}

// Parse extracts raw BPF bytecode from ELF sections.
// TODO: Implement in Iteration 1
func (p *ELFParser) Parse() ([]byte, error) {
    return nil, errors.New("ELF parser not implemented yet (Iteration 1)")
}
