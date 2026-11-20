.PHONY: fmt lint vuln test security check verify seeddev mockcert screenshots

# `go install` puts tools in GOBIN, or GOPATH/bin when GOBIN is unset, and that
# directory is not always on PATH. go_tool returns the installed copy when it
# exists and the bare name otherwise, which the shell then looks up on PATH.
GO_BIN_DIR := $(subst \,/,$(or $(shell go env GOBIN),$(shell go env GOPATH)/bin))
GOEXE := $(shell go env GOEXE)
go_tool = $(or $(wildcard $(GO_BIN_DIR)/$(1)$(GOEXE)),$(1))

GOIMPORTS ?= $(call go_tool,goimports)
GOLANGCI_LINT ?= golangci-lint
GOVULNCHECK ?= govulncheck
GOSEC ?= gosec
GITLEAKS ?= gitleaks

fmt:
	$(GOIMPORTS) -local github.com/vasfvitor/nanci -w .
	gofmt -s -w .

# The desktop app is a second Go module; each target covers both, as
# scripts/verify.sh does.
lint:
	$(GOLANGCI_LINT) run ./... && cd internal/desktop && $(GOLANGCI_LINT) run ./...

vuln:
	$(GOVULNCHECK) ./... ./internal/desktop/...

test:
	go test ./... ./internal/desktop/...

security:
	$(GOSEC) -exclude-generated ./... ./internal/desktop/...
	$(GITLEAKS) detect --source .

check: fmt vuln lint test security

# verify runs what CI checks, Go and frontend, without changing files.
verify:
	bash scripts/verify.sh

seeddev:
	go run ./cmd/seeddev

mockcert:
	go run ./cmd/mockcert

screenshots:
	cd internal/desktop/frontend && pnpm run screenshots

