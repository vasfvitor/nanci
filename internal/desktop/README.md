# Nanci Desktop

Aplicativo desktop do Nanci, em [Wails v2](https://wails.io/). É um módulo Go separado (`github.com/vasfvitor/nanci/internal/desktop`) que usa o mesmo núcleo do CLI (`internal/app`).

- **Backend** (arquivos `.go` desta pasta): `App` expõe os métodos chamados pelo frontend e devolve DTOs de `desktopapi/`, nunca structs de domínio.
- **Frontend** (`frontend/`): Vue 3, Quasar, Pinia e TypeScript. Os bindings gerados ficam em `frontend/wailsjs/` e só são importados por `frontend/src/platform/wails/`.
- **Build** (`build/`): ícones, manifesto do Windows e o script NSIS do instalador (`build/windows/installer/project.nsi`).

## Rodar

```bash
wails dev
```

Roda `pnpm install`, sobe o Vite e abre a janela com recarga automática. Requisitos e o resto do fluxo em [docs/DEVELOPMENT.md](../../docs/DEVELOPMENT.md).

## Gerar o instalador

```bash
wails build -platform windows/amd64 -nsis -m
```

Saída em `build/bin/nanci-desktop-amd64-installer.exe`.

## Regras do frontend

Páginas, stores, composables e diálogos seguem a seção "Desktop Frontend Architecture" do [AGENTS.md](../../AGENTS.md).
