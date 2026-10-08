package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/store/seed"
	"github.com/vasfvitor/nanci/internal/sync"
)

func main() {
	rootDir, err := os.Getwd()
	if err != nil {
		fatalf("get working directory: %v", err)
	}

	devdataDir := filepath.Join(rootDir, "devdata")
	if err := os.MkdirAll(devdataDir, 0o750); err != nil {
		fatalf("create devdata directory: %v", err)
	}

	dbPath := filepath.Join(devdataDir, "nanci-v1.db")
	db, err := store.OpenDB(context.Background(), dbPath, true)
	if err != nil {
		fatalf("open dev db: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Copy mock cert to devdata/certs
	certsDir := filepath.Join(devdataDir, "certs")
	if err := os.MkdirAll(certsDir, 0o750); err != nil {
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
	comp, err := seed.SeedDevelopment(ctx, db)
	if err != nil {
		fatalf("seed dev data: %v", err)
	}

	// Copy and process XMLs
	xmlDir := filepath.Join(devdataDir, "xml")
	if err := os.MkdirAll(xmlDir, 0o750); err != nil {
		fatalf("create devdata/xml directory: %v", err)
	}

	testDataDir := filepath.Join(rootDir, "internal", "nfse", "testdata")
	xmlFiles := []string{"simple-prestada.xml", "simple-tomada.xml", "com-retencoes.xml"}
	syncStore := sync.NewStore(db)
	for i, f := range xmlFiles {
		src := filepath.Join(testDataDir, f)
		dst := filepath.Join(xmlDir, f)
		if fileExists(src) {
			if err := copyFile(src, dst); err != nil {
				fatalf("copy xml: %v", err)
			}

			if err := seed.SeedDocument(ctx, syncStore, comp, dst, int64(i+1)); err != nil {
				fatalf("seed xml %s: %v", f, err)
			}
		}
	}

	fmt.Printf("Seed completed successfully.\nDatabase: %s\n", dbPath)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src) // #nosec G304 -- dev seeder reads its own fixture files.
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst) // #nosec G304 -- dev seeder writes into its own data dir.
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

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
