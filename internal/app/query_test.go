package app

import (
	"context"
	"strings"
	"testing"
)

func TestQueryNFSeRejectsInvalidAccessKeyBeforeClientSetup(t *testing.T) {
	t.Parallel()

	application := &App{}

	_, err := application.QueryNFSe(context.Background(), QueryNFSeInput{
		CNPJ:        "11222333000181",
		ChaveAcesso: strings.Repeat("1", 49) + "/",
	})
	if err == nil {
		t.Fatal("expected invalid access key error")
	}
	if !strings.Contains(err.Error(), "chave de acesso inválida") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestQueryNFSeEventsRejectsInvalidAccessKeyBeforeClientSetup(t *testing.T) {
	t.Parallel()

	application := &App{}

	_, err := application.QueryNFSeEvents(context.Background(), QueryNFSeInput{
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
