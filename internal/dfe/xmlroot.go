package dfe

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
)

// utf8BOM is the byte order mark some editors write at the start of a
// UTF-8 file.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// RootElement returns the local name (without namespace prefix) of the root
// element of a fiscal XML, such as "nfeProc" or "procEventoCTe", so a file
// that did not come with a docZip schema can be routed to its parser. A
// leading UTF-8 BOM, whitespace, the XML declaration and comments are
// skipped; any other text before the root element makes the XML invalid.
func RootElement(data []byte) (string, error) {
	data = bytes.TrimPrefix(data, utf8BOM)
	if len(bytes.TrimSpace(data)) == 0 {
		return "", errors.New("XML vazio")
	}

	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return "", errors.New("XML inválido: nenhum elemento encontrado")
		}
		if err != nil {
			return "", fmt.Errorf("XML inválido: %w", err)
		}
		switch t := token.(type) {
		case xml.StartElement:
			return t.Name.Local, nil
		case xml.CharData:
			if len(bytes.TrimSpace(t)) > 0 {
				return "", errors.New("XML inválido: texto antes do elemento raiz")
			}
		}
	}
}
