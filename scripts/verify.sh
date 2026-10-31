#!/usr/bin/env bash
# verify.sh runs on your machine what CI checks in .github/workflows/ci.yml:
# frontend lint, tests and build; Go vet, tests and lint in both modules;
# go.mod tidiness; Wails bindings drift; govulncheck when installed. Run it
# before every commit (`make verify`, or `bash scripts/verify.sh` where make
# is missing). It does not stop at the first failure: the summary at the end
# lists every step that failed. Exit code 0 means everything passed.
set -u
cd "$(dirname "$0")/.."

failed=()

step() {
	local name=$1
	shift
	echo
	echo "==> $name"
	if "$@"; then
		echo "    ok"
	else
		echo "    FALHOU: $name"
		failed+=("$name")
	fi
}

need() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "    falta a ferramenta '$1'. Instale com: $2"
		return 1
	fi
}

# go.work is gitignored; CI creates the same one.
if [ ! -f go.work ]; then
	echo "==> criando go.work"
	go work init . ./internal/desktop ./third_party/go-pkcs12
fi

frontend_deps() {
	need pnpm "corepack enable (pnpm 10, veja docs/DEVELOPMENT.md)" || return 1
	[ -d internal/desktop/frontend/node_modules ] || (cd internal/desktop/frontend && pnpm install --frozen-lockfile)
}

frontend() {
	(cd internal/desktop/frontend && pnpm run "$1")
}

lint() {
	need golangci-lint "go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2" || return 1
	golangci-lint run ./... && (cd internal/desktop && golangci-lint run ./...)
}

tidy() {
	(export GOWORK=off && go mod tidy -diff && cd internal/desktop && go mod tidy -diff)
}

# The generator analyses the Windows build the app ships, as in CI. A change
# here means a Go method or DTO changed without regenerating the bindings.
bindings() {
	need wails3 "go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0" || return 1
	(cd internal/desktop && GOOS=windows CGO_ENABLED=0 wails3 generate bindings -clean=true -ts -i >/dev/null) || return 1
	local dir=internal/desktop/frontend/bindings
	if ! git -c core.fileMode=false diff --quiet -- "$dir" || [ -n "$(git ls-files --others --exclude-standard -- "$dir")" ]; then
		echo "    os bindings mudaram: confira 'git status $dir' e commite junto com a mudança em Go"
		return 1
	fi
}

vuln() {
	if ! command -v govulncheck >/dev/null 2>&1; then
		echo "    pulado: govulncheck não está instalado (go install golang.org/x/vuln/cmd/govulncheck@v1.1.4)"
		return 0
	fi
	govulncheck ./... && (cd internal/desktop && govulncheck ./...)
}

step "dependências do frontend" frontend_deps
step "frontend: lint" frontend lint:check
step "frontend: testes" frontend test:unit
# The build writes frontend/dist, which the desktop Go module embeds.
step "frontend: build" frontend build
step "go vet" go vet ./... ./internal/desktop/...
step "go test (raiz)" go test ./...
step "go test (desktop)" go test ./internal/desktop/...
step "golangci-lint" lint
step "go.mod arrumado" tidy
step "bindings do Wails" bindings
step "govulncheck" vuln

echo
if [ ${#failed[@]} -eq 0 ]; then
	echo "Tudo certo: todas as verificações passaram."
	exit 0
fi
echo "Falharam ${#failed[@]} verificação(ões):"
for name in "${failed[@]}"; do
	echo "  - $name"
done
exit 1
