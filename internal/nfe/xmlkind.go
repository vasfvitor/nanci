package nfe

// XMLKind is the type of an NF-e XML file, known from its root element. It
// routes a file that did not come with a docZip schema (an imported XML) to
// its parser, as ClassifySchema does for the distribution.
type XMLKind string

const (
	KindProcNFe       XMLKind = "procNFe"       // nfeProc, read by ParseProcNFe
	KindResNFe        XMLKind = "resNFe"        // read by ParseResNFe
	KindProcEventoNFe XMLKind = "procEventoNFe" // read by ParseProcEventoNFe
	KindResEvento     XMLKind = "resEvento"     // read by ParseResEvento
)

// KindOfRoot maps the local name of a root element to its XMLKind. A bare
// NFe (without protNFe) is not one of them: ParseProcNFe needs the protocol.
func KindOfRoot(root string) (XMLKind, bool) {
	switch root {
	case "nfeProc":
		return KindProcNFe, true
	case "resNFe":
		return KindResNFe, true
	case "procEventoNFe":
		return KindProcEventoNFe, true
	case "resEvento":
		return KindResEvento, true
	default:
		return "", false
	}
}
