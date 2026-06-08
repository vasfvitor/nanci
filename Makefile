.PHONY: fmt lint vuln test security check

# Executables with fallback logic
GOIMPORTS ?= goimports
GOLANGCI_LINT ?= golangci-lint
GOVULNCHECK ?= govulncheck
GOSEC ?= gosec
GITLEAKS ?= gitleaks

# If goimports is not in PATH, look in GOPATH/bin/goimports
ifeq (, $(shell where $(GOIMPORTS) 2>NUL))
	GOIMPORTS_PATH = $(shell go env GOPATH)/bin/goimports.exe
	# Use fallback if file exists
	ifneq (, $(wildcard $(GOIMPORTS_PATH)))
		GOIMPORTS = $(GOIMPORTS_PATH)
	endif
endif

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
	$(GOSEC) ./...
	$(GITLEAKS) detect --source .

check: fmt vuln lint test security

seeddev:
	go run ./cmd/seeddev

mockcert:
	go run ./cmd/mockcert
