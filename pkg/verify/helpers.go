package verify

// HelperProfiles contains conservative helper index sets by program section/type.
// These are conservative and may be expanded as needed.
var HelperProfiles = map[string]map[int]bool{
	"xdp": {
		1:  true, // bpf_map_lookup_elem
		2:  true, // bpf_map_update_elem
		3:  true, // bpf_map_delete_elem
		4:  true,
		5:  true,
		6:  true,
		7:  true,
		8:  true,
		9:  true,
		10: true,
	},
	"socket": {
		1: true,
		2: true,
		3: true,
	},
	"generic": {
		1:  true,
		2:  true,
		3:  true,
		4:  true,
		5:  true,
		6:  true,
		7:  true,
		8:  true,
		9:  true,
		10: true,
		11: true,
		12: true,
		13: true,
		14: true,
		15: true,
		16: true,
		17: true,
		18: true,
		19: true,
		20: true,
	},
}

// AllowedHelpersForSection returns conservative allowed helper map for a given section name.
func AllowedHelpersForSection(section string) map[int]bool {
	if section == "" {
		return HelperProfiles["generic"]
	}

	// simple matching
	if section == "xdp" || len(section) >= 4 && section[:4] == "xdp/" {
		return HelperProfiles["xdp"]
	}
	if section == "socket" || len(section) >= 7 && section[:7] == "socket/" {
		return HelperProfiles["socket"]
	}

	return HelperProfiles["generic"]
}
