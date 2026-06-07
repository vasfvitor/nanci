package nfse

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// ParseEventXML parses a National NFS-e event XML using strict, namespace-aware path tracking.
// It supports both XSD 1.00 and 1.01 element structures.
func ParseEventXML(data []byte) (Event, []string, error) {
	var ev Event
	var warnings []string

	ev.Type = EventType("unknown") // Default to unknown
	ev.CreatedAt = time.Now().UTC()

	decoder := xml.NewDecoder(bytes.NewReader(data))
	
	if len(bytes.TrimSpace(data)) == 0 {
		return ev, nil, errors.New("empty xml event")
	}

	var pathStack []string
	var currentText strings.Builder

	// Flags to detect specific event blocks
	var isCancelamento, isSubstituicao bool

	for {
		t, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return ev, nil, fmt.Errorf("xml parse error: %w", err)
		}

		switch se := t.(type) {
		case xml.StartElement:
			local := se.Name.Local
			pathStack = append(pathStack, local)
			currentText.Reset()

			if local == "infPedCanc" || local == "infCanc" || local == "infPedidoCanc" {
				isCancelamento = true
			} else if local == "infSubst" || local == "Substituicao" || local == "substituicao" {
				isSubstituicao = true
			}

		case xml.CharData:
			currentText.Write(se)

		case xml.EndElement:
			val := strings.TrimSpace(currentText.String())
			currentPath := strings.Join(pathStack, "/")
			
			if len(pathStack) > 0 {
				pathStack = pathStack[:len(pathStack)-1]
			}
			
			if val == "" {
				continue
			}

			// We use suffix matching for paths
			if strings.HasSuffix(currentPath, "/chNFSe") || strings.HasSuffix(currentPath, "/chNFSePed") {
				parsed, err := ParseAccessKey(val)
				if err != nil {
					return ev, nil, fmt.Errorf("invalid event access key: %w", err)
				}
				ev.ChaveAcesso = parsed
			} else if strings.HasSuffix(currentPath, "/dhEvento") || strings.HasSuffix(currentPath, "/dhRegEvento") || strings.HasSuffix(currentPath, "/dhProc") {
				if parsed, err := time.Parse(time.RFC3339, val); err == nil {
					ev.EventAt = parsed
					ev.EventAtValid = true
				} else {
					warnings = append(warnings, fmt.Sprintf("invalid time format: %s", val))
				}
			} else if strings.HasSuffix(currentPath, "/chSubstituta") || strings.HasSuffix(currentPath, "/chNFSeSubst") || strings.HasSuffix(currentPath, "/chNFSeSubstituida") {
				ev.ReplacementChaveAcesso = val
			} else if strings.HasSuffix(currentPath, "/cMotivo") || strings.HasSuffix(currentPath, "/xMotivo") {
				ev.Description = val
			}

			currentText.Reset()
		}
	}

	// Classify event type
	if isCancelamento {
		ev.Type = EventType("cancelamento")
	} else if isSubstituicao || ev.ReplacementChaveAcesso != "" {
		ev.Type = EventType("substituicao")
	} else {
		warnings = append(warnings, "unsupported or unknown event structure")
	}

	if ev.ChaveAcesso == "" {
		return ev, nil, errors.New("missing essential field: chNFSe")
	}

	if ev.Description == "" {
		ev.Description = "Evento sincronizado via NSU"
	}

	ev.ParseWarnings = warnings
	return ev, warnings, nil
}
