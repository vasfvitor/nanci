// Package fiscalrenderer implements dacte.Renderer with
// github.com/awafinance/fiscal-renderer (LGPL-3.0).
package fiscalrenderer

import (
	"bytes"
	"fmt"

	frdacte "github.com/awafinance/fiscal-renderer/dacte"

	"github.com/vasfvitor/nanci/internal/dacte"
)

type adapter struct{}

// New returns a DACTE renderer backed by github.com/awafinance/fiscal-renderer.
func New() dacte.Renderer {
	return adapter{}
}

func (adapter) Render(xml []byte, opts dacte.Options) ([]byte, error) {
	doc, err := frdacte.New(string(xml), &frdacte.Config{WatermarkCancelled: opts.Cancelada})
	if err != nil {
		return nil, fmt.Errorf("ler XML do CT-e: %w", err)
	}
	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		return nil, fmt.Errorf("gerar DACTE: %w", err)
	}
	return buf.Bytes(), nil
}

var _ dacte.Renderer = adapter{}
