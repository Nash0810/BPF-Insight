package verify

import (
    "strings"
)

// Rule groups
const (
    RuleGroupPointer     = "pointer"
    RuleGroupStack       = "stack"
    RuleGroupMap         = "map"
    RuleGroupControlFlow = "controlflow"
    RuleGroupHelper      = "helper"
    RuleGroupGeneral     = "general"
)

type Rule struct {
    ID       string
    Group    string
    Severity int    // 1=info, 2=warn, 3=high, 4=critical
    Enabled  bool
}

// Registry of all rules used by BPFInsight
var RuleRegistry = map[string]*Rule{
    "pointer_arithmetic": {
        ID: "pointer_arithmetic", Group: RuleGroupPointer, Severity: 3, Enabled: true,
    },
    "write_r10": {
        ID: "write_r10", Group: RuleGroupPointer, Severity: 4, Enabled: true,
    },
    "stack_var_offset": {
        ID: "stack_var_offset", Group: RuleGroupStack, Severity: 3, Enabled: true,
    },
    "map_no_nullcheck": {
        ID: "map_no_nullcheck", Group: RuleGroupMap, Severity: 3, Enabled: true,
    },
    "map_update_nocheck": {
        ID: "map_update_nocheck", Group: RuleGroupMap, Severity: 3, Enabled: true,
    },
    "helper_chain": {
        ID: "helper_chain", Group: RuleGroupHelper, Severity: 2, Enabled: true,
    },
    "unknown_jump": {
        ID: "unknown_jump", Group: RuleGroupControlFlow, Severity: 3, Enabled: true,
    },
    "high_complexity": {
        ID: "high_complexity", Group: RuleGroupControlFlow, Severity: 3, Enabled: true,
    },
}

// Apply profile (disable all rules, then re-enable only profile rules)
func ApplyProfile(name string) {
    for _, r := range RuleRegistry {
        r.Enabled = false
    }

    if rules, ok := Profiles[name]; ok {
        for _, id := range rules {
            if rule, exists := RuleRegistry[id]; exists {
                rule.Enabled = true
            }
        }
    }
}

// Enable specific rules (comma separated)
func EnableRules(list string) {
    if list == "" {
        return
    }

    for _, id := range strings.Split(list, ",") {
        id = strings.TrimSpace(id)
        if rule, ok := RuleRegistry[id]; ok {
            rule.Enabled = true
        }
    }
}

// Disable specific rules (comma separated)
func DisableRules(list string) {
    if list == "" {
        return
    }

    for _, id := range strings.Split(list, ",") {
        id = strings.TrimSpace(id)
        if rule, ok := RuleRegistry[id]; ok {
            rule.Enabled = false
        }
    }
}
