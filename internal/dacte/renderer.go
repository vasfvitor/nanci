// Package dacte defines how nanci renders the DACTE, the printed form of a
// CT-e, from its stored XML.
package dacte

// Options tunes one DACTE.
type Options struct {
	// Cancelada prints the cancellation watermark.
	Cancelada bool
}

// Renderer generates a DACTE PDF from the authorized CT-e XML bytes
// (procCTe or a bare CTe). Only CT-e modelo 57 has a layout; see
// cte.Document.HasDACTE.
type Renderer interface {
	Render(xml []byte, opts Options) ([]byte, error)
}
