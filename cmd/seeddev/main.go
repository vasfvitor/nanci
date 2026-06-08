package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/store/seed"
)

func main() {
	rootDir, err := os.Getwd()
	if err != nil {
		fatalf("get working directory: %v", err)
	}

	devdataDir := filepath.Join(rootDir, "devdata")
	if err := os.MkdirAll(devdataDir, 0o755); err != nil {
		fatalf("create devdata directory: %v", err)
	}

	dbPath := filepath.Join(devdataDir, "nanci-v2.db")
	db, err := store.OpenDB(dbPath, true)
	if err != nil {
		fatalf("open dev db: %v", err)
	}
	defer db.Close()

	// Copy mock cert to devdata/certs
	certsDir := filepath.Join(devdataDir, "certs")
	if err := os.MkdirAll(certsDir, 0o755); err != nil {
		fatalf("create devdata/certs directory: %v", err)
	}

	srcCert := filepath.Join(rootDir, "internal", "foundation", "cert", "testdata", "cert_a1_mock_70860312000150.pfx")
	dstCert := filepath.Join(certsDir, "cert_a1_mock_70860312000150.pfx")
	if fileExists(srcCert) {
		if err := copyFile(srcCert, dstCert); err != nil {
			fatalf("copy mock cert: %v", err)
		}
	} else {
		info("warning: mock cert not found at %s. Please run cmd/mockcert first if you want it copied.", srcCert)
	}

	ctx := context.Background()
	if err := seed.SeedDevelopment(ctx, db); err != nil {
		fatalf("seed dev data: %v", err)
	}

	// Copy and process XMLs
	xmlDir := filepath.Join(devdataDir, "xml")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		fatalf("create devdata/xml directory: %v", err)
	}

	testDataDir := filepath.Join(rootDir, "internal", "nfse", "testdata")
	xmlFiles := []string{"simple-prestada.xml", "simple-tomada.xml", "com-retencoes.xml"}
	for _, f := range xmlFiles {
		src := filepath.Join(testDataDir, f)
		dst := filepath.Join(xmlDir, f)
		if fileExists(src) {
			if err := copyFile(src, dst); err != nil {
				fatalf("copy xml: %v", err)
			}
			
			// Process and insert
			if err := seedXML(ctx, db, dst, "dev-company-70860312000150"); err != nil {
				fatalf("seed xml %s: %v", f, err)
			}
		}
	}

	fmt.Printf("Seed completed successfully.\nDatabase: %s\n", dbPath)
}

func seedXML(ctx context.Context, db *sql.DB, xmlPath, companyID string) error {
	data, err := os.ReadFile(xmlPath)
	if err != nil {
		return err
	}

	doc, _, err := nfse.ParseDocumentXML(data)
	if err != nil {
		return err
	}

	doc.ID = nfse.DocumentID("doc-" + doc.ChaveAcesso)
	doc.XMLPath = xmlPath
	doc.RawHash = "hash-" + string(doc.ChaveAcesso)
	
	if err := seed.UpsertDocument(ctx, db, doc); err != nil {
		return err
	}

	role := nfse.CompanyRole("none")
	if doc.TomadorCNPJ == "70860312000150" {
		role = nfse.CompanyRoleTomada
	} else if doc.PrestadorCNPJ == "70860312000150" {
		role = nfse.CompanyRolePrestada
	}

	cd := nfse.CompanyDocument{
		Document:   doc,
		RelationID: fmt.Sprintf("%s-%s", companyID, doc.ID),
		CompanyID:  nfse.CompanyID(companyID),
		DocumentID: doc.ID,
		CompanyRole: role,
		VisibilityReason: nfse.VisibilityReason("unknown"),
	}

	return seed.UpsertCompanyDocument(ctx, db, cd)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func info(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
