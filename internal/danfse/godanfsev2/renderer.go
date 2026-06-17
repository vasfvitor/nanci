package godanfsev2

import (
	"regexp"

	danfsev2 "github.com/wevertonj/go-danfse-v2"

	"github.com/vasfvitor/nanci/internal/danfse"
)

var nfseRegex = regexp.MustCompile(`(?i)(<nfse[\s>][\s\S]*?</nfse>)`)

type adapter struct {
	inner danfse.Renderer
}

// New returns a DANFSe renderer backed by github.com/wevertonj/go-danfse-v2.
func New() danfse.Renderer {
	return &adapter{
		inner: danfsev2.NewDanfseRenderer(),
	}
}

func (a *adapter) Render(xmlData []byte) ([]byte, error) {
	if match := nfseRegex.Find(xmlData); match != nil {
		xmlData = match
	}
	return a.inner.Render(xmlData)
}

var _ danfse.Renderer = (*adapter)(nil)
