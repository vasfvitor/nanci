.PHONY: fmt lint vuln test security check seeddev mockcert screenshots

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

lint:
	$(GOLANGCI_LINT) run ./...

vuln:
	$(GOVULNCHECK) ./...

test:
	go test ./...

security:
	$(GOSEC) -exclude-generated ./...
	$(GITLEAKS) detect --source .

check: fmt vuln lint test security

seeddev:
	go run ./cmd/seeddev

mockcert:
	go run ./cmd/mockcert

screenshots:
	cd internal/desktop/frontend && pnpm run screenshots

