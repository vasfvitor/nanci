package nfe

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
)

// walkXML streams data with encoding/xml, keeping a stack of local element
// names, the same technique as nfse/document_parser.go. onStart is called for
// every start element and onText for every element with non-blank text. Both
// receive the path from the root with a leading slash, for example
// "/nfeProc/NFe/infNFe/ide/nNF". Namespace prefixes are ignored. Parsers match
// paths by suffix, anchored on enough parent elements to tell apart fields that
// share a name, such as total/ICMSTot/vICMS and the per-item vICMS.
func walkXML(data []byte, onStart func(path string, attrs []xml.Attr), onText func(path, text string) error) error {
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

// hasAnySuffix reports whether path ends with one of the suffixes.
func hasAnySuffix(path string, suffixes ...string) bool {
	for _, s := range suffixes {
		if strings.HasSuffix(path, s) {
			return true
		}
	}
	return false
}

func attrValue(attrs []xml.Attr, name string) string {
	for _, a := range attrs {
		if a.Name.Local == name {
			return strings.TrimSpace(a.Value)
		}
	}
	return ""
}

// parseDateTime parses the NF-e date-time format (RFC 3339 with offset). On
// failure it appends a warning naming the field and returns nil.
func parseDateTime(field, value string, warnings *[]string) *time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		*warnings = append(*warnings, fmt.Sprintf("invalid %s format: %s", field, value))
		return nil
	}
	return &parsed
}

// competence returns "YYYY-MM" in the offset dhEmi was written with, or ""
// for a zero time.
func competence(issueDate time.Time) string {
	if issueDate.IsZero() {
		return ""
	}
	return issueDate.Format("2006-01")
}

// withoutLeadingZeros turns the zero-padded serie and nNF slots of the access
// key into the form used inside the NF-e ("001" -> "1", "000" -> "0").
func withoutLeadingZeros(s string) string {
	trimmed := strings.TrimLeft(s, "0")
	if trimmed == "" && s != "" {
		return "0"
	}
	return trimmed
}

// parseMoneyInto parses an XSD decimal into dst, naming field in the error.
func parseMoneyInto(dst *nfse.Money, field, value string) error {
	m, err := nfse.ParseMoney(value)
	if err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	*dst = m
	return nil
}
