package danfse

// Renderer generates a DANFSe PDF from the authorized NFS-e XML bytes.
//
// The contract intentionally stays at XML-in/PDF-out because DANFSe layout is
// defined by national technical notes and renderer libraries may use very
// different internal models.
type Renderer interface {
	Render(xml []byte) ([]byte, error)
}
