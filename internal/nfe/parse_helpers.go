package nfe

import "strings"

// withoutLeadingZeros turns the zero-padded serie and nNF slots of the access
// key into the form used inside the NF-e ("001" -> "1", "000" -> "0").
func withoutLeadingZeros(s string) string {
	trimmed := strings.TrimLeft(s, "0")
	if trimmed == "" && s != "" {
		return "0"
	}
	return trimmed
}
