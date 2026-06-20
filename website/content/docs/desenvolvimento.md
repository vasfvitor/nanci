---
title: "Desenvolvimento e Contribuição"
description: "Como configurar o ambiente local, rodar e compilar o Nanci a partir do código-fonte."
weight: 90
icon: "code"
toc: true
---

O Nanci é composto por duas frentes principais:
1. **CLI (`cmd/nanci`)**: Aplicação terminal compilada puramente em Go.
2. **Desktop (`internal/desktop`)**: Aplicação visual construída com [Wails](https://wails.io/), Go (Backend) e Vue 3 (Frontend).

---

## Requisitos

Para desenvolver ou compilar o projeto localmente, você precisará de:

- Go 1.23+
- Node.js 24+
- pnpm (gerenciador de pacotes do Node)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation/)
- Compilador C/C++ (como `mingw-w64` ou `gcc`) para compilar as partes Cgo do Wails e SQLite.

Instale o Wails CLI:
```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

---

## Ambiente Local (Go Workspace)

Para facilitar a resolução de pacotes (como o módulo interno `internal/desktop` e pacotes em `third_party`), recomendamos criar um arquivo `go.work` na raiz do projeto (ele já é ignorado pelo `.gitignore`):

```go
go 1.26.3

use (
	.
	./internal/desktop
	./third_party/go-pkcs12
)
```

Isso resolve problemas de IDE (como VSCode/GoLand) ao navegar no código.

---

## Ambiente Local (Dev Data)

Para evitar sujar o seu diretório pessoal com dados de teste, o projeto possui um script para provisionar um banco de dados temporário e injetar certificados falsos de teste.

Rode o script na raiz do projeto:
```bash
make seeddev
```
*Ou `go run ./cmd/seeddev` se não tiver o utilitário Make instalado.*

Isso criará a pasta `devdata/` contendo:
- `nanci-dev.db` (banco SQLite com tabelas criadas).
- `certs/` (certificados de mock).

Você pode gerar novos certificados de mock a qualquer momento com:
```bash
make mockcert
```
*(Requer OpenSSL instalado na máquina).*

---

## Rodando a Aplicação

### CLI
Para compilar e rodar o CLI:
```bash
go build -o nanci.exe ./cmd/nanci
./nanci.exe --help
```

### Desktop App (Wails)
Para desenvolver com hot-reload (Live Reload do Vue e re-compilação rápida do Go):
```bash
cd internal/desktop
wails dev
```
O servidor frontend irá iniciar, e uma janela nativa aparecerá. Salvar arquivos em `frontend/src` atualizará a tela automaticamente.

---

## Compilando para Produção

Para gerar o instalador do Windows:
```bash
cd internal/desktop
wails build -platform windows/amd64 -nsis -m
```
O instalador final estará localizado em `internal/desktop/build/bin/`.

---

## Captura Automatizada de Telas (Screenshots)

Para manter a documentação e o `README.md` atualizados com as telas mais recentes do aplicativo sem precisar inserir dados reais manualmente, o projeto possui um gerador automatizado de capturas de tela. Ele utiliza o **Vite** e o **Playwright** para simular o backend e renderizar o frontend com dados fictícios estruturados, gerando capturas em alta resolução.

Para executar o gerador a partir da raiz do projeto, use:

```bash
make screenshots
```

Ou no Windows (PowerShell):

```powershell
.\make.ps1 screenshots
```

Os arquivos gerados são salvos em `docs/screenshots/` nos temas claro e escuro.
