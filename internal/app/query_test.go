package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/store/storetest"
)

func TestQueryNFSeEventsRejectsInvalidAccessKeyBeforeClientSetup(t *testing.T) {
	t.Parallel()

	application := &App{}

	_, err := application.Query.QueryNFSeEvents(context.Background(), QueryNFSeInput{
		CNPJ:        "11222333000181",
		ChaveAcesso: strings.Repeat("1", 49) + "?",
	})
	if err == nil {
		t.Fatal("expected invalid access key error")
	}
	if !strings.Contains(err.Error(), "chave de acesso inválida") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateQueryAccessKeyAcceptsAlphanumericCNPJ(t *testing.T) {
	t.Parallel()

	const chave = "355030812" + "12ABC34501DE35" + "0000000000123" + "2608" + "123456789" + "7"
	got, err := validateQueryAccessKey(" " + strings.ToLower(chave) + " ")
	if err != nil {
		t.Fatalf("validateQueryAccessKey: %v", err)
	}
	if string(got) != chave {
		t.Errorf("validateQueryAccessKey = %q, want %q", got, chave)
	}
}

func TestTestConnectionReportsUnknownCompanyOnce(t *testing.T) {
	t.Parallel()
	env := newNFeTestEnv(t)

	_, err := env.app.Query.TestConnection(context.Background(), "11222333000181")
	if err == nil {
		t.Fatal("expected error for an unregistered CNPJ")
	}
	if got := strings.Count(err.Error(), "empresa não encontrada"); got != 1 {
		t.Fatalf("error mentions %q %d times, want 1: %v", "empresa não encontrada", got, err)
	}
}

// TestTestConnectionCredentialLookup: a missing credential is a diagnostic
// in the result; a database error is returned.
func TestTestConnectionCredentialLookup(t *testing.T) {
	t.Parallel()

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		env := newNFeTestEnv(t)
		env.app.Query.CredentialStore = credential.NewStore(storetest.OpenTestDB(t)) // no credentials

		result, err := env.app.Query.TestConnection(context.Background(), nfeTestCNPJ)
		if err != nil {
			t.Fatalf("TestConnection: %v", err)
		}
		if result.CertLoaded || !strings.Contains(result.StatusExplanation, "não associado ou não encontrado") {
			t.Errorf("result = %+v, want the missing certificate explanation", result)
		}
	})

	t.Run("database error", func(t *testing.T) {
		t.Parallel()
		env := newNFeTestEnv(t)
		closed := storetest.OpenTestDB(t)
		if err := closed.Close(); err != nil {
			t.Fatal(err)
		}
		env.app.Query.CredentialStore = credential.NewStore(closed)

		result, err := env.app.Query.TestConnection(context.Background(), nfeTestCNPJ)
		if err == nil || !strings.HasPrefix(err.Error(), "buscar credencial: ") {
			t.Fatalf("TestConnection error = %v, want a wrapped credential lookup error", err)
		}
		if errors.Is(err, credential.ErrCredentialNotFound) || result.StatusExplanation != "" {
			t.Errorf("database error reported as a missing credential: %v, %+v", err, result)
		}
		if len(env.passwords.requests) != 0 {
			t.Errorf("password prompts = %d, want 0", len(env.passwords.requests))
		}
	})
}
