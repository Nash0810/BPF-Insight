package parser

import (
	"debug/elf"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ELFParser loads eBPF bytecode from ELF files.
type ELFParser struct {
	FilePath string
}

// Parse extracts raw BPF bytecode from ELF sections.
func (p *ELFParser) Parse() ([]byte, error) {
	if p.FilePath == "" {
		return nil, errors.New("no ELF file path provided")
	}

	f, err := os.Open(p.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ELF file: %w", err)
	}
	defer f.Close()

	ef, err := elf.NewFile(f)
	if err != nil {
		return nil, fmt.Errorf("invalid ELF file: %w", err)
	}

	// Candidate section names for eBPF code
	sectionPrefixes := []string{
		"xdp",
		"tracepoint/",
		"kprobe/",
		"kretprobe/",
		"raw_tracepoint/",
		"socket",
		".text",
	}

	var codeSection *elf.Section

	for _, sec := range ef.Sections {
		secName := sec.Name

		for _, prefix := range sectionPrefixes {
			if strings.HasPrefix(secName, prefix) {
				codeSection = sec
				break
			}
		}
	}

	if codeSection == nil {
		return nil, fmt.Errorf("no eBPF program section found in ELF file: %s", p.FilePath)
	}

	// Read eBPF bytecode
	data, err := codeSection.Data()
	if err != nil {
		return nil, fmt.Errorf("failed to read section data: %w", err)
	}

	return data, nil
}

// ParseELF extracts raw BPF bytecode and section name from ELF files.
// Returns: (raw bytecode, section name, error)
func ParseELF(filePath string) ([]byte, string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open ELF file: %w", err)
	}
	defer f.Close()

	ef, err := elf.NewFile(f)
	if err != nil {
		return nil, "", fmt.Errorf("invalid ELF file: %w", err)
	}

	// Candidate section names for eBPF code
	sectionPrefixes := []string{
		"xdp",
		"tracepoint/",
		"kprobe/",
		"kretprobe/",
		"raw_tracepoint/",
		"socket",
		".text",
	}

	var codeSection *elf.Section
	var sectionName string

	for _, sec := range ef.Sections {
		secName := sec.Name

		for _, prefix := range sectionPrefixes {
			if strings.HasPrefix(secName, prefix) {
				codeSection = sec
				sectionName = secName
				break
			}
		}
		if codeSection != nil {
			break
		}
	}

	if codeSection == nil {
		return nil, "", fmt.Errorf("no eBPF program section found in ELF file: %s", filePath)
	}

	// Read eBPF bytecode
	data, err := codeSection.Data()
	if err != nil {
		return nil, "", fmt.Errorf("failed to read section data: %w", err)
	}

	return data, sectionName, nil
}