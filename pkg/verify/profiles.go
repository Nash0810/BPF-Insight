package verify

// Profile definitions
// These specify which rules are turned on in each profile.
var Profiles = map[string][]string{

    // Balanced default
    "default": {
        "pointer_arithmetic",
        "write_r10",
        "stack_var_offset",
        "map_no_nullcheck",
        "helper_chain",
        "unknown_jump",
        "high_complexity",
    },

    // Strict: more aggressive, for debugging complex kernels
    "strict": {
        "pointer_arithmetic",
        "write_r10",
        "stack_var_offset",
        "map_no_nullcheck",
        "map_update_nocheck",
        "helper_chain",
        "unknown_jump",
        "high_complexity",
    },

    // Kernel-like verifier rigidity
    "kernel": {
        "pointer_arithmetic",
        "write_r10",
        "map_no_nullcheck",
        "map_update_nocheck",
        "unknown_jump",
        "high_complexity",
    },

    // Very forgiving
    "relaxed": {
        "write_r10",
        "map_no_nullcheck",
    },
}
