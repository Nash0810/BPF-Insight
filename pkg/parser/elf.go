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

// isBPFSection checks if a section name is a valid eBPF program section.
func isBPFSection(name string) bool {
	if name == "xdp" || strings.HasPrefix(name, "xdp/") {
		return true
	}
	if strings.HasPrefix(name, "kprobe/") {
		return true
	}
	if strings.HasPrefix(name, "kretprobe/") {
		return true
	}
	if strings.HasPrefix(name, "tracepoint/") {
		return true
	}
	if strings.HasPrefix(name, "raw_tracepoint/") {
		return true
	}
	if name == "socket" || strings.HasPrefix(name, "socket/") {
		return true
	}
	// BPF test programs compiled with clang always use .text
	if name == ".text" || strings.HasPrefix(name, ".text") {
		return true
	}
	return false
}

// findBPFSection finds the first non-empty BPF section, prioritizing explicit sections over .text
func findBPFSection(ef *elf.File) *elf.Section {
	// First pass: look for explicit BPF sections (xdp, kprobe, etc.)
	explicitPrefixes := []string{"xdp", "kprobe", "kretprobe", "tracepoint", "raw_tracepoint", "socket"}
	for _, prefix := range explicitPrefixes {
		for _, sec := range ef.Sections {
			if (sec.Name == prefix || strings.HasPrefix(sec.Name, prefix+"/")) && sec.Size > 0 {
				return sec
			}
		}
	}

	// Second pass: look for any .text section
	for _, sec := range ef.Sections {
		if (sec.Name == ".text" || strings.HasPrefix(sec.Name, ".text")) && sec.Size > 0 {
			return sec
		}
	}

	// Fallback: return first non-empty BPF section
	for _, sec := range ef.Sections {
		if isBPFSection(sec.Name) && sec.Size > 0 {
			return sec
		}
	}

	return nil
}

// Parse extracts raw BPF bytecode from the first eBPF program section.
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

	var codeSection *elf.Section

	codeSection = findBPFSection(ef)

	if codeSection == nil {
		return nil, fmt.Errorf("no eBPF program section found in ELF file: %s", p.FilePath)
	}

	data, err := codeSection.Data()
	if err != nil {
		return nil, fmt.Errorf("failed to read ELF section data: %w", err)
	}

	return data, nil
}

// ParseELF returns raw bytecode + section name.
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

	var codeSection *elf.Section
	var sectionName string

	codeSection = findBPFSection(ef)

	if codeSection == nil {
		return nil, "", fmt.Errorf("no eBPF section found in ELF: %s", filePath)
	}

	sectionName = codeSection.Name

	data, err := codeSection.Data()
	if err != nil {
		return nil, "", fmt.Errorf("failed to read ELF section data: %w", err)
	}

	return data, sectionName, nil
}

// HasBTF checks whether the ELF contains a .BTF section
func HasBTF(filePath string) (bool, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	ef, err := elf.NewFile(f)
	if err != nil {
		return false, err
	}

	for _, sec := range ef.Sections {
		if sec == nil {
			continue
		}
		if sec.Name == ".BTF" || strings.HasPrefix(sec.Name, ".BTF") {
			return true, nil
		}
	}
	return false, nil
}
