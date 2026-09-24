package cte

import "strings"

// SchemaKind is the payload type of a docZip, taken from its schema attribute.
type SchemaKind int

const (
	SchemaUnknown SchemaKind = iota
	SchemaProcCTe
	SchemaProcCTeOS
	SchemaProcGTVe
	SchemaProcCTeSimp
	SchemaProcEventoCTe
)

// ClassifySchema reads the name before "_v" in a docZip schema attribute,
// such as "procCTe_v4.00.xsd" or "procEventoCTe_v4.00.xsd". Unknown names
// return SchemaUnknown so the caller can store the payload without parsing
// it.
func ClassifySchema(schema string) SchemaKind {
	name, _, _ := strings.Cut(strings.TrimSpace(schema), "_v")
	switch name {
	case "procCTe":
		return SchemaProcCTe
	case "procCTeOS":
		return SchemaProcCTeOS
	case "procGTVe":
		return SchemaProcGTVe
	case "procCTeSimp":
		return SchemaProcCTeSimp
	case "procEventoCTe":
		return SchemaProcEventoCTe
	default:
		return SchemaUnknown
	}
}

func (k SchemaKind) String() string {
	switch k {
	case SchemaProcCTe:
		return "procCTe"
	case SchemaProcCTeOS:
		return "procCTeOS"
	case SchemaProcGTVe:
		return "procGTVe"
	case SchemaProcCTeSimp:
		return "procCTeSimp"
	case SchemaProcEventoCTe:
		return "procEventoCTe"
	default:
		return "unknown"
	}
}
