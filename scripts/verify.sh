#!/usr/bin/env bash
# verify.sh runs on your machine what .github/workflows/ci.yml checks, with
# the tool versions read from that file. It runs every step and lists the
# failures at the end; exit code 0 means everything passed. What each step
# does: docs/DEVELOPMENT.md, "Verificar antes do PR".
set -u
cd "$(dirname "$0")/.."

ci=.github/workflows/ci.yml
wails_version=$(sed -nE 's/^ *WAILS_VERSION: *(v[0-9.]+) *$/\1/p' "$ci" | head -1)
sqlc_version=$(sed -nE 's/^ *SQLC_VERSION: *(v[0-9.]+) *$/\1/p' "$ci" | head -1)
lint_version=$(sed -nE 's/^ *version: *(v[0-9.]+) *$/\1/p' "$ci" | head -1)
vuln_version=$(sed -nE 's/.*govulncheck@(v[0-9.]+).*/\1/p' "$ci" | head -1)

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

# tool checks that the installed tool is the version CI uses.
tool() {
	local name=$1 want=$2 install=$3
	if ! command -v "$name" >/dev/null 2>&1; then
		echo "    falta '$name' $want. Instale com: $install"
		return 1
	fi
	if ! "$name" version 2>&1 | grep -qF "${want#v}"; then
		echo "    '$name' não está na versão $want do CI. Instale com: $install"
		return 1
	fi
}

# go.work is gitignored; CI creates the same one.
if [ ! -f go.work ]; then
	echo "==> criando go.work"
	go work init . ./internal/desktop ./third_party/go-pkcs12
fi

frontend_deps() {
	if ! command -v pnpm >/dev/null 2>&1; then
		echo "    falta 'pnpm' (versão 10, veja docs/DEVELOPMENT.md)"
		return 1
	fi
	[ -d internal/desktop/frontend/node_modules ] || (cd internal/desktop/frontend && pnpm install --frozen-lockfile)
}

frontend() {
	(cd internal/desktop/frontend && pnpm run "$1")
}

lint() {
	tool golangci-lint "$lint_version" "go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$lint_version" || return 1
	golangci-lint run ./... && (cd internal/desktop && golangci-lint run ./...)
}

tidy() {
	(export GOWORK=off && go mod tidy -diff && cd internal/desktop && go mod tidy -diff)
}

# The generator analyses the Windows build the app ships, as in CI.
bindings() {
	tool wails3 "$wails_version" "go install github.com/wailsapp/wails/v3/cmd/wails3@$wails_version" || return 1
	(cd internal/desktop && GOOS=windows CGO_ENABLED=0 wails3 generate bindings -clean=true -ts -i >/dev/null) || return 1
	local dir=internal/desktop/frontend/bindings
	if ! git -c core.fileMode=false diff --quiet -- "$dir" || [ -n "$(git ls-files --others --exclude-standard -- "$dir")" ]; then
		echo "    os bindings mudaram: confira 'git status $dir' e commite junto com a mudança em Go"
		return 1
	fi
}

sqlc() {
	CGO_ENABLED=0 go run "github.com/sqlc-dev/sqlc/cmd/sqlc@$sqlc_version" diff || {
		echo "    internal/store/sqlgen não bate com as queries e migrations: regenere (skill migration)"
		return 1
	}
}

step "dependências do frontend" frontend_deps
step "frontend: lint" frontend lint:check
step "frontend: testes" frontend test:unit
# The build writes frontend/dist, which the desktop Go module embeds.
step "frontend: build" frontend build
step "go test" go test ./... ./internal/desktop/...
step "golangci-lint" lint
step "go.mod arrumado" tidy
step "bindings do Wails" bindings
step "código do sqlc" sqlc
step "govulncheck" go run "golang.org/x/vuln/cmd/govulncheck@$vuln_version" ./... ./internal/desktop/...

# A change to the rules is not a failure, but whoever asked for the work
# must approve it: the agent that changed them cannot be the one to judge.
rules=$(
	base=$(git merge-base HEAD main 2>/dev/null) || exit 0
	git diff --name-only "$base" -- AGENTS.md .claude/skills .golangci.yml .github/workflows scripts/verify.sh \
		internal/desktop/frontend/eslint.config.ts internal/desktop/architecture_test.go internal/store/migrations_frozen_test.go
	git diff --name-only --diff-filter=MDR "$base" -- internal/store/migrations_v2
)

echo
if [ -n "$rules" ]; then
	echo "Atenção: estes arquivos de regra mudaram desde a main. Confirme com quem pediu o trabalho:"
	echo "$rules" | sed 's/^/  - /'
	echo
fi
if [ ${#failed[@]} -eq 0 ]; then
	echo "Tudo certo: todas as verificações passaram."
	exit 0
fi
echo "Falharam ${#failed[@]} verificação(ões):"
for name in "${failed[@]}"; do
	echo "  - $name"
done
exit 1
