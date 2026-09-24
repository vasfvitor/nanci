// Package xmlwalk streams an XML document and reports each element by its
// path from the root, so fiscal document parsers can match fields by path
// suffix instead of declaring a struct for every layout version.
package xmlwalk

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Walk streams data with encoding/xml, keeping a stack of local element names.
// onStart, when not nil, is called for every start element and onText for
// every element with non-blank text. Both receive the path from the root with
// a leading slash, for example "/nfeProc/NFe/infNFe/ide/nNF". Namespace
// prefixes are ignored. Parsers match paths by suffix, anchored on enough
// parent elements to tell apart fields that share a name, such as
// total/ICMSTot/vICMS and the per-item vICMS. An error returned by onText
// stops the walk and is returned as is.
func Walk(data []byte, onStart func(path string, attrs []xml.Attr), onText func(path, text string) error) error {
	if len(bytes.TrimSpace(data)) == 0 {
		return errors.New("empty xml document")
	}

	decoder := xml.NewDecoder(bytes.NewReader(data))
	var stack []string
	var text strings.Builder

	for {
		tok, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("xml parse error: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			stack = append(stack, t.Name.Local)
			text.Reset()
			if onStart != nil {
				onStart("/"+strings.Join(stack, "/"), t.Attr)
			}
		case xml.CharData:
			text.Write(t)
		case xml.EndElement:
			path := "/" + strings.Join(stack, "/")
			stack = stack[:len(stack)-1]
			value := strings.TrimSpace(text.String())
			text.Reset()
			if value == "" {
				continue
			}
			if err := onText(path, value); err != nil {
				return err
			}
		}
	}
}

// HasAnySuffix reports whether path ends with one of the suffixes.
func HasAnySuffix(path string, suffixes ...string) bool {
	for _, s := range suffixes {
		if strings.HasSuffix(path, s) {
			return true
		}
	}
	return false
}

// AttrValue returns the trimmed value of the attribute with the given local
// name, or an empty string when it is absent.
func AttrValue(attrs []xml.Attr, name string) string {
	for _, a := range attrs {
		if a.Name.Local == name {
			return strings.TrimSpace(a.Value)
		}
	}
	return ""
}
