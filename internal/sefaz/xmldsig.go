// Portions adapted from github.com/mschunke/gonfe (MIT License). See third_party/gonfe/LICENSE.

package sefaz

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1" // #nosec G505 -- the NF-e XMLDSig profile mandates RSA-SHA1.
	"crypto/tls"
	"encoding/base64"
	"errors"
)

const xmldsigNamespace = "http://www.w3.org/2000/09/xmldsig#"

// Signer signs eventos with the XMLDSig profile of the NF-e: enveloped
// signature over infEvento, inclusive C14N 1.0, SHA-1 digest, RSA-SHA1
// signature and the leaf certificate in KeyInfo.
type Signer struct {
	key     *rsa.PrivateKey
	certDER []byte
}

// NewSigner takes the certificate loaded by foundation/cert.LoadPKCS12. It
// needs an RSA private key and the parsed leaf.
func NewSigner(cert tls.Certificate) (*Signer, error) {
	key, ok := cert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("the certificate private key is not RSA")
	}
	if cert.Leaf == nil {
		return nil, errors.New("the certificate has no parsed leaf")
	}
	return &Signer{key: key, certDER: cert.Leaf.Raw}, nil
}

// SignEvento returns the standalone signed evento:
//
//	<evento xmlns="http://www.portalfiscal.inf.br/nfe" versao="1.00"><infEvento Id="…">…</infEvento><Signature …>…</Signature></evento>
//
// infEvento is written already canonical, so no C14N library is needed: its
// digest input is the same bytes with the inherited NF-e namespace declared.
func (s *Signer) SignEvento(e Evento, tpAmb string) ([]byte, error) {
	id, infEvento, err := buildInfEvento(e, tpAmb)
	if err != nil {
		return nil, err
	}

	canonicalInfEvento := `<infEvento xmlns="` + nfeNamespace + `"` + infEvento[len("<infEvento"):]
	digest := sha1.Sum([]byte(canonicalInfEvento)) // #nosec G401 -- the NF-e XMLDSig profile mandates SHA-1.
	digestValue := base64.StdEncoding.EncodeToString(digest[:])

	signed := sha1.Sum([]byte(signedInfo(id, digestValue, true))) // #nosec G401 -- the NF-e XMLDSig profile mandates SHA-1.
	signature, err := rsa.SignPKCS1v15(nil, s.key, crypto.SHA1, signed[:])
	if err != nil {
		return nil, err
	}

	evento := `<evento xmlns="` + nfeNamespace + `" versao="` + versaoEvento + `">` +
		infEvento +
		`<Signature xmlns="` + xmldsigNamespace + `">` +
		signedInfo(id, digestValue, false) +
		`<SignatureValue>` + base64.StdEncoding.EncodeToString(signature) + `</SignatureValue>` +
		`<KeyInfo><X509Data><X509Certificate>` + base64.StdEncoding.EncodeToString(s.certDER) + `</X509Certificate></X509Data></KeyInfo>` +
		`</Signature></evento>`
	return []byte(evento), nil
}

// signedInfo writes SignedInfo in canonical form. The signature is computed
// over the element with the xmldsig namespace declared (withNamespace); the
// copy inside Signature inherits it and leaves the declaration out.
func signedInfo(id, digestValue string, withNamespace bool) string {
	open := `<SignedInfo>`
	if withNamespace {
		open = `<SignedInfo xmlns="` + xmldsigNamespace + `">`
	}
	return open +
		`<CanonicalizationMethod Algorithm="http://www.w3.org/TR/2001/REC-xml-c14n-20010315"></CanonicalizationMethod>` +
		`<SignatureMethod Algorithm="http://www.w3.org/2000/09/xmldsig#rsa-sha1"></SignatureMethod>` +
		`<Reference URI="#` + id + `">` +
		`<Transforms>` +
		`<Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"></Transform>` +
		`<Transform Algorithm="http://www.w3.org/TR/2001/REC-xml-c14n-20010315"></Transform>` +
		`</Transforms>` +
		`<DigestMethod Algorithm="http://www.w3.org/2000/09/xmldsig#sha1"></DigestMethod>` +
		`<DigestValue>` + digestValue + `</DigestValue>` +
		`</Reference></SignedInfo>`
}
