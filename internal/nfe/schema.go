package nfe

import "strings"

// SchemaKind is the payload type of a docZip, taken from its schema attribute.
type SchemaKind int

const (
	SchemaUnknown SchemaKind = iota
	SchemaResNFe
	SchemaProcNFe
	SchemaResEvento
	SchemaProcEventoNFe
)

// ClassifySchema reads the name before "_v" in a docZip schema attribute,
// such as "resNFe_v1.01.xsd" or "procNFe_v4.00.xsd". Unknown names return
// SchemaUnknown so the caller can store the payload without parsing it.
func ClassifySchema(schema string) SchemaKind {
	name, _, _ := strings.Cut(strings.TrimSpace(schema), "_v")
	switch name {
	case "resNFe":
		return SchemaResNFe
	case "procNFe":
		return SchemaProcNFe
	case "resEvento":
		return SchemaResEvento
	case "procEventoNFe":
		return SchemaProcEventoNFe
	default:
		return SchemaUnknown
	}
}

func (k SchemaKind) String() string {
	switch k {
	case SchemaResNFe:
		return "resNFe"
	case SchemaProcNFe:
		return "procNFe"
	case SchemaResEvento:
		return "resEvento"
	case SchemaProcEventoNFe:
		return "procEventoNFe"
	default:
		return "unknown"
	}
}
