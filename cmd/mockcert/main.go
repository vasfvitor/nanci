package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	cnpj         = "70860312000150"
	password     = "mockdata"
	validityDays = 365

	outputDirName = "internal/foundation/cert/testdata"
	pfxFileName   = "cert_a1_mock_70860312000150.pfx"
)

func main() {
	rootDir, err := os.Getwd()
	if err != nil {
		fatalf("get working directory: %v", err)
	}

	outputDir := filepath.Join(rootDir, outputDirName)
	pfxFile := filepath.Join(outputDir, pfxFileName)

	opensslPath, err := findOpenSSL()
	if err != nil {
		fatalf("openssl not found; install it or add it to PATH")
	}

	version, err := runCommand(opensslPath, "version")
	if err != nil {
		fatalf("openssl check failed: %v\n%s", err, version)
	}
	info("using %s", strings.TrimSpace(version))

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fatalf("create output directory: %v", err)
	}

	tempDir, err := os.MkdirTemp("", "nanci-mock-cert-*")
	if err != nil {
		fatalf("create temp directory: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	cnfFile := filepath.Join(tempDir, "cert_config.cnf")
	keyFile := filepath.Join(tempDir, "cert.key")
	crtFile := filepath.Join(tempDir, "cert.crt")

	if err := os.WriteFile(cnfFile, []byte(opensslConfig()), 0o600); err != nil {
		fatalf("write openssl config: %v", err)
	}

	if fileExists(pfxFile) {
		info("pfx already exists: %s", pfxFile)
		return
	}

	out, err := runCommand(opensslPath, "genrsa", "-out", keyFile, "2048")
	if err != nil || !fileExists(keyFile) {
		fatalf("generate private key: %v\n%s", err, out)
	}

	out, err = runCommand(
		opensslPath,
		"req",
		"-new",
		"-x509",
		"-key", keyFile,
		"-out", crtFile,
		"-days", fmt.Sprintf("%d", validityDays),
		"-config", cnfFile,
		"-extensions", "v3_ext",
		"-utf8",
	)
	if err != nil || !fileExists(crtFile) {
		fatalf("generate certificate: %v\n%s", err, out)
	}

	out, err = runCommand(
		opensslPath,
		"pkcs12",
		"-export",
		"-certpbe", "PBE-SHA1-3DES",
		"-keypbe", "PBE-SHA1-3DES",
		"-macalg", "sha1",
		"-out", pfxFile,
		"-inkey", keyFile,
		"-in", crtFile,
		"-name", fmt.Sprintf("Certificado A1 Mock - CNPJ %s", cnpj),
		"-passout", "pass:"+password,
	)
	if err != nil || !fileExists(pfxFile) {
		fatalf("generate pfx: %v\n%s", err, out)
	}

	certInfo, err := runCommand(opensslPath, "x509", "-in", crtFile, "-noout", "-subject", "-dates", "-serial")
	if err != nil {
		fatalf("read certificate info: %v\n%s", err, certInfo)
	}

	fmt.Printf("pfx=%s\n", pfxFile)
	fmt.Printf("cnpj=%s\n", formatCNPJ(cnpj))
	fmt.Printf("password=%s\n", password)
	fmt.Printf("validity_days=%d\n", validityDays)

	for _, line := range strings.Split(strings.TrimSpace(certInfo), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			fmt.Println(line)
		}
	}
}

func opensslConfig() string {
	return fmt.Sprintf(`oid_section = new_oids

[ new_oids ]
icp_brasil_cnpj = 2.16.76.1.3.3

[ req ]
default_bits       = 2048
prompt             = no
default_md         = sha256
distinguished_name = dn
x509_extensions    = v3_ext
string_mask        = utf8only

[ dn ]
C  = BR
ST = SP
L  = Sao Paulo
O  = Empresa Mock Teste
OU = Testes
CN = Certificado A1 Mock %[1]s
icp_brasil_cnpj = %[1]s

[ v3_ext ]
basicConstraints = critical,CA:FALSE
keyUsage = critical,digitalSignature,keyEncipherment
extendedKeyUsage = clientAuth
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid,issuer
subjectAltName = @alt_names

[ alt_names ]
otherName.1 = icp_brasil_cnpj;UTF8:%[1]s
`, cnpj)
}

func findOpenSSL() (string, error) {
	if path, err := exec.LookPath("openssl"); err == nil {
		return path, nil
	}

	if runtime.GOOS != "windows" {
		return "", errors.New("openssl not found")
	}

	candidates := []string{
		`C:\Program Files\OpenSSL-Win64\bin\openssl.exe`,
		`C:\Program Files\OpenSSL-Win32\bin\openssl.exe`,
		`C:\Program Files (x86)\OpenSSL-Win32\bin\openssl.exe`,
	}

	for _, candidate := range candidates {
		if fileExists(candidate) {
			return candidate, nil
		}
	}

	return "", errors.New("openssl not found")
}

func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err := cmd.Run()
	return output.String(), err
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func formatCNPJ(value string) string {
	if len(value) != 14 {
		return value
	}

	return fmt.Sprintf(
		"%s.%s.%s/%s-%s",
		value[0:2],
		value[2:5],
		value[5:8],
		value[8:12],
		value[12:14],
	)
}

func info(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}