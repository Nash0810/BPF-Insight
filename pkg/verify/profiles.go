package verify

import "fmt"

type Profile struct {
	Name        string
	Description string
	Enable      []string
}

var Profiles = map[string]Profile{
	"default": {
		Name:        "default",
		Description: "Balanced defaults for static analysis",
		Enable: []string{
			"ptr", "stack", "r10", "helpers",
			"nullcheck", "mapkey", "complex",
		},
	},

	"strict": {
		Name:        "strict",
		Description: "Kernel-like strictness",
		Enable: []string{
			"ptr", "stack", "r10", "helpers",
			"nullcheck", "mapkey", "complex",
			"jump", "fanout", "ctx",
		},
	},

	"safe": {
		Name:        "safe",
		Description: "Safety-focused minimal rule set",
		Enable: []string{
			"ptr", "r10", "nullcheck", "mapkey",
		},
	},
}

// Apply profile by name
func ApplyProfile(name string) error {
	p, ok := Profiles[name]
	if !ok {
		return fmt.Errorf("unknown profile: %s", name)
	}

	// Disable all
	for _, r := range Rules {
		r.Enabled = false
	}

	// Enable selected
	for _, rule := range p.Enable {
		if long, ok := RuleAliases[rule]; ok {
			rule = long
		}
		r, ok := Rules[rule]
		if !ok {
			return fmt.Errorf("profile %s references unknown rule: %s", name, rule)
		}
		r.Enabled = true
	}

	return nil
}
