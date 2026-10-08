package app

import (
	"context"
	"strings"
	"testing"
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
