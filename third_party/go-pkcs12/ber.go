package pkcs12

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"
)

const maxASN1Depth = 64

type asn1Object interface {
	encodeTo(*bytes.Buffer) error
}

type asn1Structured struct {
	tagBytes []byte
	content  []asn1Object
}

func (s asn1Structured) encodeTo(out *bytes.Buffer) error {
	inner := new(bytes.Buffer)
	for _, obj := range s.content {
		if err := obj.encodeTo(inner); err != nil {
			return err
		}
	}
	if _, err := out.Write(s.tagBytes); err != nil {
		return err
	}
	if err := encodeLength(out, inner.Len()); err != nil {
		return err
	}
	_, err := out.Write(inner.Bytes())
	return err
}

type asn1Primitive struct {
	tagBytes []byte
	content  []byte
}

func (p asn1Primitive) encodeTo(out *bytes.Buffer) error {
	if _, err := out.Write(p.tagBytes); err != nil {
		return err
	}
	if err := encodeLength(out, len(p.content)); err != nil {
		return err
	}
	_, err := out.Write(p.content)
	return err
}

type berParser struct {
	data    []byte
	changed bool
}

// normalizeBER converts supported BER forms into DER. Primitive contents are
// preserved; constructed OCTET STRING chunks are concatenated.
func normalizeBER(ber []byte) ([]byte, error) {
	if len(ber) == 0 {
		return nil, errors.New("ber2der: input BER is empty")
	}

	parser := berParser{data: ber}
	obj, offset, err := parser.readObject(0, 0)
	if err != nil {
		return nil, err
	}
	if offset != len(ber) {
		return nil, errors.New("ber2der: trailing data after ASN.1 object")
	}

	out := new(bytes.Buffer)
	if err := obj.encodeTo(out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func (p *berParser) readObject(offset, depth int) (asn1Object, int, error) {
	if depth > maxASN1Depth {
		return nil, 0, errors.New("ber2der: ASN.1 nesting is too deep")
	}
	if offset >= len(p.data) {
		return nil, 0, errors.New("ber2der: truncated ASN.1 tag")
	}

	tagStart := offset
	firstTagByte := p.data[offset]
	offset++
	class := firstTagByte >> 6
	constructed := firstTagByte&0x20 != 0
	tag := int(firstTagByte & 0x1f)
	if tag == 0x1f {
		tag = 0
		first := true
		for {
			if offset >= len(p.data) {
				return nil, 0, errors.New("ber2der: truncated high-tag-number")
			}
			b := p.data[offset]
			offset++
			if first && b == 0x80 {
				return nil, 0, errors.New("ber2der: non-minimal high-tag-number")
			}
			first = false
			if tag > (math.MaxInt-int(b&0x7f))/128 {
				return nil, 0, errors.New("ber2der: ASN.1 tag number is too large")
			}
			tag = tag*128 + int(b&0x7f)
			if b&0x80 == 0 {
				break
			}
		}
		if tag < 0x1f {
			return nil, 0, errors.New("ber2der: non-minimal high-tag-number")
		}
	}
	tagEnd := offset

	if class == 0 && tag == 0 {
		return nil, 0, errors.New("ber2der: unexpected end-of-content marker")
	}
	if offset >= len(p.data) {
		return nil, 0, errors.New("ber2der: truncated ASN.1 length")
	}

	firstLengthByte := p.data[offset]
	offset++
	indefinite := firstLengthByte == 0x80
	length := 0
	if firstLengthByte > 0x80 {
		numberOfBytes := int(firstLengthByte & 0x7f)
		if numberOfBytes == 0 || numberOfBytes > strconv.IntSize/8 {
			return nil, 0, errors.New("ber2der: ASN.1 length is too large")
		}
		if numberOfBytes > len(p.data)-offset {
			return nil, 0, errors.New("ber2der: truncated ASN.1 length")
		}
		if p.data[offset] == 0 {
			p.changed = true
		}
		for i := 0; i < numberOfBytes; i++ {
			if length > (math.MaxInt-int(p.data[offset]))/256 {
				return nil, 0, errors.New("ber2der: ASN.1 length overflows int")
			}
			length = length*256 + int(p.data[offset])
			offset++
		}
		if length < 128 {
			p.changed = true
		}
	} else if !indefinite {
		length = int(firstLengthByte)
	}

	if indefinite && !constructed {
		return nil, 0, errors.New("ber2der: indefinite length requires constructed encoding")
	}
	if indefinite {
		p.changed = true
	}
	if constructed && class == 0 && tag != 4 && tag != 16 && tag != 17 {
		return nil, 0, fmt.Errorf("ber2der: unsupported constructed universal tag %d", tag)
	}

	tagBytes := append([]byte(nil), p.data[tagStart:tagEnd]...)
	if !constructed {
		if length > len(p.data)-offset {
			return nil, 0, errors.New("ber2der: ASN.1 content exceeds available data")
		}
		contentEnd := offset + length
		return asn1Primitive{
			tagBytes: tagBytes,
			content:  append([]byte(nil), p.data[offset:contentEnd]...),
		}, contentEnd, nil
	}

	contentEnd := len(p.data)
	if !indefinite {
		if length > len(p.data)-offset {
			return nil, 0, errors.New("ber2der: ASN.1 content exceeds available data")
		}
		contentEnd = offset + length
	}

	var children []asn1Object
	for {
		if indefinite {
			if offset+2 > len(p.data) {
				return nil, 0, errors.New("ber2der: missing end-of-content marker")
			}
			if p.data[offset] == 0 && p.data[offset+1] == 0 {
				offset += 2
				break
			}
		} else if offset == contentEnd {
			break
		} else if offset > contentEnd {
			return nil, 0, errors.New("ber2der: child exceeds constructed content")
		}

		child, next, err := p.readObject(offset, depth+1)
		if err != nil {
			return nil, 0, err
		}
		if next <= offset || (!indefinite && next > contentEnd) {
			return nil, 0, errors.New("ber2der: invalid constructed content boundary")
		}
		children = append(children, child)
		offset = next
	}

	if class == 0 && tag == 4 {
		content, err := flattenOctetStrings(children)
		if err != nil {
			return nil, 0, err
		}
		tagBytes[0] &^= 0x20
		p.changed = true
		return asn1Primitive{tagBytes: tagBytes, content: content}, offset, nil
	}

	return asn1Structured{tagBytes: tagBytes, content: children}, offset, nil
}

func flattenOctetStrings(children []asn1Object) ([]byte, error) {
	out := new(bytes.Buffer)
	for _, child := range children {
		primitive, ok := child.(asn1Primitive)
		if !ok || len(primitive.tagBytes) != 1 || primitive.tagBytes[0] != 0x04 {
			return nil, errors.New("ber2der: constructed OCTET STRING contains a non-OCTET STRING child")
		}
		if _, err := out.Write(primitive.content); err != nil {
			return nil, err
		}
	}
	return out.Bytes(), nil
}

func encodeLength(out *bytes.Buffer, length int) error {
	if length < 0 {
		return errors.New("ber2der: negative length")
	}
	if length < 128 {
		return out.WriteByte(byte(length))
	}

	numberOfBytes := 0
	for value := length; value > 0; value >>= 8 {
		numberOfBytes++
	}
	if err := out.WriteByte(0x80 | byte(numberOfBytes)); err != nil {
		return err
	}
	for shift := (numberOfBytes - 1) * 8; shift >= 0; shift -= 8 {
		if err := out.WriteByte(byte(length >> uint(shift))); err != nil {
			return err
		}
	}
	return nil
}
