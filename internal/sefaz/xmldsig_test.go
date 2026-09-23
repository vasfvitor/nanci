package sefaz

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/foundation/cert"
	"github.com/vasfvitor/nanci/internal/nfe"
)

var update = flag.Bool("update", false, "rewrite golden files in testdata")

const testChave = "35260911222333000181550010000012341123456787"

func loadMockCertificate(t *testing.T) tls.Certificate {
	t.Helper()
	loaded, err := cert.LoadPKCS12(filepath.Join("..", "foundation", "cert", "testdata", "cert_a1_mock_70860312000150.pfx"), []byte("mockdata"))
	if err != nil {
		t.Fatalf("LoadPKCS12: %v", err)
	}
	return loaded.TLS
}

func newMockSigner(t *testing.T) *Signer {
	t.Helper()
	signer, err := NewSigner(loadMockCertificate(t))
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	return signer
}

func cienciaEvento() Evento {
	return Evento{
		ChaveAcesso: testChave,
		CNPJ:        testCNPJ,
		TpEvento:    nfe.ManifestationCiencia.TpEvento(),
		DescEvento:  nfe.ManifestationCiencia.DescEvento(),
		NSeqEvento:  1,
		DhEvento:    time.Date(2026, 9, 23, 10, 30, 58, 0, time.FixedZone("", -3*60*60)),
	}
}

func TestSignEvento_Golden(t *testing.T) {
	signed, err := newMockSigner(t).SignEvento(cienciaEvento(), TpAmbProducao)
	if err != nil {
		t.Fatalf("SignEvento: %v", err)
	}

	golden := filepath.Join("testdata", "evento-ciencia-signed.xml")
	if *update {
		if err := os.WriteFile(golden, signed, 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(golden) // #nosec G304 -- fixed testdata path.
	if err != nil {
		t.Fatalf("read golden (run with -update to create it): %v", err)
	}
	if !bytes.Equal(signed, want) {
		t.Errorf("signed evento differs from %s:\n got %s\nwant %s", golden, signed, want)
	}

	prefix := `<evento xmlns="http://www.portalfiscal.inf.br/nfe" versao="1.00">` +
		`<infEvento Id="ID21021035260911222333000181550010000012341123456787` + `01">` +
		`<cOrgao>91</cOrgao><tpAmb>1</tpAmb><CNPJ>70860312000150</CNPJ>` +
		`<chNFe>35260911222333000181550010000012341123456787</chNFe>` +
		`<dhEvento>2026-09-23T10:30:58-03:00</dhEvento><tpEvento>210210</tpEvento>` +
		`<nSeqEvento>1</nSeqEvento><verEvento>1.00</verEvento>` +
		`<detEvento versao="1.00"><descEvento>Ciencia da Operacao</descEvento></detEvento></infEvento>` +
		`<Signature xmlns="http://www.w3.org/2000/09/xmldsig#"><SignedInfo><CanonicalizationMethod`
	if !strings.HasPrefix(string(signed), prefix) {
		t.Errorf("signed evento does not start with the expected infEvento:\n%s", signed)
	}
	if !strings.HasSuffix(string(signed), `</X509Certificate></X509Data></KeyInfo></Signature></evento>`) {
		t.Errorf("Signature must be the last child of evento:\n%s", signed)
	}
}

func TestSignEvento_IndependentVerification(t *testing.T) {
	tlsCert := loadMockCertificate(t)
	signer, err := NewSigner(tlsCert)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}

	justificada := cienciaEvento()
	justificada.TpEvento = nfe.ManifestationNaoRealizada.TpEvento()
	justificada.DescEvento = nfe.ManifestationNaoRealizada.DescEvento()
	justificada.XJust = `Mercadoria "devolvida" & recusada <sem nota> 'x'`

	for name, e := range map[string]Evento{"ciencia": cienciaEvento(), "nao realizada": justificada} {
		t.Run(name, func(t *testing.T) {
			signed, err := signer.SignEvento(e, TpAmbHomologacao)
			if err != nil {
				t.Fatalf("SignEvento: %v", err)
			}
			verifySignedEvento(t, signed, tlsCert.Leaf)
		})
	}

	signed, err := signer.SignEvento(justificada, TpAmbHomologacao)
	if err != nil {
		t.Fatalf("SignEvento: %v", err)
	}
	wantDet := `<detEvento versao="1.00"><descEvento>Operacao nao Realizada</descEvento>` +
		`<xJust>Mercadoria "devolvida" &amp; recusada &lt;sem nota&gt; 'x'</xJust></detEvento>`
	if !strings.Contains(string(signed), wantDet) {
		t.Errorf("xJust must follow descEvento with C14N escaping:\n%s", signed)
	}
}

type parsedSignedEvento struct {
	InfEvento struct {
		ID string `xml:"Id,attr"`
	} `xml:"infEvento"`
	Signature struct {
		SignedInfo struct {
			Reference struct {
				URI         string `xml:"URI,attr"`
				DigestValue string `xml:"DigestValue"`
			} `xml:"Reference"`
		} `xml:"SignedInfo"`
		SignatureValue  string `xml:"SignatureValue"`
		X509Certificate string `xml:"KeyInfo>X509Data>X509Certificate"`
	} `xml:"Signature"`
}

// verifySignedEvento checks a signed evento without the production signer:
// it canonicalizes infEvento and SignedInfo from the parsed document, checks
// the digest and verifies the RSA-SHA1 signature with the leaf public key.
func verifySignedEvento(t *testing.T, signed []byte, leaf *x509.Certificate) {
	t.Helper()
	var parsed parsedSignedEvento
	if err := xml.Unmarshal(signed, &parsed); err != nil {
		t.Fatalf("parse signed evento: %v", err)
	}
	if parsed.Signature.SignedInfo.Reference.URI != "#"+parsed.InfEvento.ID {
		t.Errorf("Reference URI = %q, want #%s", parsed.Signature.SignedInfo.Reference.URI, parsed.InfEvento.ID)
	}

	canonicalInf := canonicalize(t, signed, "infEvento", nfeNamespace)
	if embedded := strings.Replace(canonicalInf, ` xmlns="`+nfeNamespace+`"`, "", 1); !strings.Contains(string(signed), embedded) {
		t.Errorf("infEvento is not embedded in canonical form:\ncanonical %s\n   signed %s", embedded, signed)
	}
	digest := sha1.Sum([]byte(canonicalInf))
	if got := base64.StdEncoding.EncodeToString(digest[:]); got != parsed.Signature.SignedInfo.Reference.DigestValue {
		t.Errorf("DigestValue = %s, recomputed %s", parsed.Signature.SignedInfo.Reference.DigestValue, got)
	}

	certDER, err := base64.StdEncoding.DecodeString(parsed.Signature.X509Certificate)
	if err != nil {
		t.Fatalf("decode X509Certificate: %v", err)
	}
	if !bytes.Equal(certDER, leaf.Raw) {
		t.Error("X509Certificate is not the leaf certificate")
	}
	signature, err := base64.StdEncoding.DecodeString(parsed.Signature.SignatureValue)
	if err != nil {
		t.Fatalf("decode SignatureValue: %v", err)
	}
	signedInfoSum := sha1.Sum([]byte(canonicalize(t, signed, "SignedInfo", xmldsigNamespace)))
	publicKey, ok := leaf.PublicKey.(*rsa.PublicKey)
	if !ok {
		t.Fatal("leaf public key is not RSA")
	}
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA1, signedInfoSum[:], signature); err != nil {
		t.Errorf("SignatureValue does not verify: %v", err)
	}
}

// The real ciência from the sped-nfe docs was accepted by SEFAZ; its digest
// proves the canonical form nanci writes is the one SEFAZ computes.
func TestInfEventoDigest_RealEvent(t *testing.T) {
	const wantDigest = "1Vki32HS6hFd/GMo6KZYBdP297s="
	realEvent, err := os.ReadFile(filepath.Join("..", "nfe", "testdata", "proceventonfe-ciencia-real.xml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	digest := sha1.Sum([]byte(canonicalize(t, realEvent, "infEvento", nfeNamespace)))
	if got := base64.StdEncoding.EncodeToString(digest[:]); got != wantDigest {
		t.Fatalf("test canonicalizer digest = %s, want %s", got, wantDigest)
	}

	dhEvento, err := time.Parse(time.RFC3339, "2017-03-18T10:30:58-03:00")
	if err != nil {
		t.Fatal(err)
	}
	id, infEvento, err := buildInfEvento(Evento{
		ChaveAcesso: "35170349607369000156550010000229481398694060",
		CNPJ:        "69161982000108",
		TpEvento:    nfe.TpEventoCiencia,
		DescEvento:  nfe.ManifestationCiencia.DescEvento(),
		NSeqEvento:  1,
		DhEvento:    dhEvento,
	}, TpAmbProducao)
	if err != nil {
		t.Fatalf("buildInfEvento: %v", err)
	}
	if id != "ID2102103517034960736900015655001000022948139869406001" {
		t.Errorf("Id = %s", id)
	}
	if !bytes.Contains(realEvent, []byte(infEvento)) {
		t.Errorf("built infEvento is not byte-identical to the real one:\n%s", infEvento)
	}
	built := sha1.Sum([]byte(`<infEvento xmlns="` + nfeNamespace + `"` + strings.TrimPrefix(infEvento, "<infEvento")))
	if got := base64.StdEncoding.EncodeToString(built[:]); got != wantDigest {
		t.Errorf("built infEvento digest = %s, want %s", got, wantDigest)
	}
}

func TestSignEvento_Validation(t *testing.T) {
	signer := newMockSigner(t)
	tests := map[string]func(*Evento){
		"invalid key":            func(e *Evento) { e.ChaveAcesso = testChave[:43] + "0" },
		"invalid CNPJ":           func(e *Evento) { e.CNPJ = "70860312000151" },
		"invalid tpEvento":       func(e *Evento) { e.TpEvento = "2102" },
		"nSeqEvento zero":        func(e *Evento) { e.NSeqEvento = 0 },
		"nSeqEvento too big":     func(e *Evento) { e.NSeqEvento = 21 },
		"missing dhEvento":       func(e *Evento) { e.DhEvento = time.Time{} },
		"missing descEvento":     func(e *Evento) { e.DescEvento = "" },
		"xJust on ciência":       func(e *Evento) { e.XJust = "justificativa qualquer" },
		"missing xJust":          func(e *Evento) { e.TpEvento = nfe.TpEventoNaoRealizada },
		"xJust with whitespace":  func(e *Evento) { e.TpEvento = nfe.TpEventoNaoRealizada; e.XJust = " operacao nao realizada" },
		"xJust with line break":  func(e *Evento) { e.TpEvento = nfe.TpEventoNaoRealizada; e.XJust = "operacao\r\nnao realizada" },
		"descEvento not UTF-8":   func(e *Evento) { e.DescEvento = "Ciencia\xff" },
		"descEvento with a tab":  func(e *Evento) { e.DescEvento = "Ciencia\tda Operacao" },
		"descEvento with spaces": func(e *Evento) { e.DescEvento = "Ciencia da Operacao " },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			e := cienciaEvento()
			mutate(&e)
			if _, err := signer.SignEvento(e, TpAmbProducao); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestNewSigner(t *testing.T) {
	tlsCert := loadMockCertificate(t)

	withoutLeaf := tlsCert
	withoutLeaf.Leaf = nil
	if _, err := NewSigner(withoutLeaf); err == nil {
		t.Error("expected error without a parsed leaf")
	}

	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	withECKey := tlsCert
	withECKey.PrivateKey = ecKey
	if _, err := NewSigner(withECKey); err == nil {
		t.Error("expected error for a non-RSA key")
	}
}

func TestFormatDhEvento(t *testing.T) {
	utc := time.Date(2026, 9, 23, 13, 30, 58, 999, time.UTC)
	if got := formatDhEvento(utc); got != "2026-09-23T10:30:58-03:00" {
		t.Errorf("formatDhEvento = %s", got)
	}
}

// canonicalize writes the first element named local in doc, with its
// subtree, as C14N 1.0 without comments. It declares ns as the default
// namespace on that element and drops every other namespace declaration,
// which is exact for NF-e documents: one default namespace per subtree and
// no prefixes. It deliberately shares no code with the signer.
func canonicalize(t *testing.T, doc []byte, local, ns string) string {
	t.Helper()
	d := xml.NewDecoder(bytes.NewReader(doc))
	var b strings.Builder
	depth := 0
	for {
		tok, err := d.RawToken()
		if errors.Is(err, io.EOF) {
			t.Fatalf("element %s not found", local)
		}
		if err != nil {
			t.Fatalf("canonicalize: %v", err)
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			if depth == 0 && tok.Name.Local != local {
				continue
			}
			b.WriteString("<" + tok.Name.Local)
			if depth == 0 {
				b.WriteString(` xmlns="` + ns + `"`)
			}
			var attrs []xml.Attr
			for _, a := range tok.Attr {
				if a.Name.Space == "xmlns" || (a.Name.Space == "" && a.Name.Local == "xmlns") {
					continue
				}
				attrs = append(attrs, a)
			}
			sort.Slice(attrs, func(i, j int) bool { return attrs[i].Name.Local < attrs[j].Name.Local })
			for _, a := range attrs {
				b.WriteString(" " + a.Name.Local + `="` + c14nAttr(a.Value) + `"`)
			}
			b.WriteString(">")
			depth++
		case xml.EndElement:
			if depth == 0 {
				continue
			}
			b.WriteString("</" + tok.Name.Local + ">")
			depth--
			if depth == 0 {
				return b.String()
			}
		case xml.CharData:
			if depth > 0 {
				b.WriteString(c14nText(string(tok)))
			}
		}
	}
}

func c14nText(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\r", "&#xD;").Replace(s)
}

func c14nAttr(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", `"`, "&quot;", "\t", "&#x9;", "\n", "&#xA;", "\r", "&#xD;").Replace(s)
}
