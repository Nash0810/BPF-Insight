package verify

// Mapping of short profile names → actual rule names in registry
var RuleAliases = map[string]string{
	"ptr":       "pointer_arithmetic",
	"stack":     "stack_var_offset",
	"r10":       "write_r10",
	"helpers":   "helper_chain",
	"nullcheck": "map_no_null_check",
	"mapkey":    "map_update_no_key_check",
	"complex":   "high_complexity",

	// Reserved for future iterations
	"jump":   "unknown_jump",
	"fanout": "branch_fanout",
	"ctx":    "ctx_mod",
}
