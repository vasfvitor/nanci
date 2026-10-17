package cte

// XMLKind is the type of a CT-e XML file, known from its root element. It
// routes a file that did not come with a docZip schema (an imported XML) to
// its parser, as ClassifySchema does for the distribution.
type XMLKind string

const (
	// KindProcCTe is any authorized transport document: cteProc, cteOSProc,
	// GTVeProc or cteSimpProc, all read by ParseProcCTe.
	KindProcCTe XMLKind = "procCTe"
	// KindProcEventoCTe is a procEventoCTe, read by ParseProcEventoCTe.
	KindProcEventoCTe XMLKind = "procEventoCTe"
)

// KindOfRoot maps the local name of a root element to its XMLKind.
func KindOfRoot(root string) (XMLKind, bool) {
	switch root {
	case "cteProc", "cteOSProc", "GTVeProc", "cteSimpProc":
		return KindProcCTe, true
	case "procEventoCTe":
		return KindProcEventoCTe, true
	default:
		return "", false
	}
}
