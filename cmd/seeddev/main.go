package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

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

	dbPath := filepath.Join(devdataDir, "nanci-dev.db")
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

	fmt.Printf("Seed completed successfully.\nDatabase: %s\n", dbPath)
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
