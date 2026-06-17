package godanfsev2

import (
	danfsev2 "github.com/wevertonj/go-danfse-v2"

	"github.com/vasfvitor/nanci/internal/danfse"
)

// New returns a DANFSe renderer backed by github.com/wevertonj/go-danfse-v2.
func New() danfse.Renderer {
	return danfsev2.NewDanfseRenderer()
}

var _ danfse.Renderer = danfsev2.NewDanfseRenderer()
