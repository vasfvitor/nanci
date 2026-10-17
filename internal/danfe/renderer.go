// Package danfe defines how nanci renders the DANFE, the printed form of an
// NF-e, from its stored XML.
package danfe

// Options tunes one DANFE.
type Options struct {
	// Cancelada prints the cancellation watermark.
	Cancelada bool
}

// Renderer generates a DANFE PDF from the authorized NF-e XML bytes
// (procNFe or a bare NFe).
type Renderer interface {
	Render(xml []byte, opts Options) ([]byte, error)
}
