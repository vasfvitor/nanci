// Package fiscalrenderer implements danfe.Renderer with
// github.com/awafinance/fiscal-renderer (LGPL-3.0).
package fiscalrenderer

import (
	"bytes"
	"fmt"

	frdanfe "github.com/awafinance/fiscal-renderer/danfe"

	"github.com/vasfvitor/nanci/internal/danfe"
)

type adapter struct{}

// New returns a DANFE renderer backed by github.com/awafinance/fiscal-renderer.
func New() danfe.Renderer {
	return adapter{}
}

func (adapter) Render(xml []byte, opts danfe.Options) ([]byte, error) {
	doc, err := frdanfe.New(string(xml), &frdanfe.Config{WatermarkCancelled: opts.Cancelada})
	if err != nil {
		return nil, fmt.Errorf("ler XML da NF-e: %w", err)
	}
	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		return nil, fmt.Errorf("gerar DANFE: %w", err)
	}
	return buf.Bytes(), nil
}

var _ danfe.Renderer = adapter{}
