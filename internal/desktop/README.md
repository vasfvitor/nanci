# Nanci Desktop

Aplicativo desktop do Nanci, em [Wails v3](https://v3.wails.io/) (v3.0.0). É um módulo Go separado (`github.com/vasfvitor/nanci/internal/desktop`) que usa o mesmo núcleo do CLI (`internal/app`).

- **Backend** (arquivos `.go` desta pasta): `App` é o único serviço do Wails, expõe os métodos chamados pelo frontend e devolve DTOs de `desktopapi/`, nunca structs de domínio.
- **Frontend** (`frontend/`): Vue 3, Quasar, Pinia e TypeScript. Os bindings gerados ficam em `frontend/bindings/` e, como o `@wailsio/runtime`, só são importados por `frontend/src/platform/wails/`.
- **Build** (`build/`): `config.yml` do Wails, os Taskfiles, ícones, manifesto do Windows e o script NSIS do instalador (`build/windows/nsis/project.nsi`).

## Rodar

```bash
wails3 dev
```

Compila o app em modo dev, sobe o Vite em `127.0.0.1:9245` e abre a janela com recarga automática. Requisitos e o resto do fluxo em [docs/DEVELOPMENT.md](../../docs/DEVELOPMENT.md).

## Gerar o instalador

```bash
wails3 task windows:package
```

Saída em `bin/nanci-desktop-amd64-installer.exe` (o executável fica em `bin/nanci-desktop.exe`). Requer o NSIS (`makensis`) no PATH.

## Regras do frontend

Páginas, stores, composables e diálogos seguem a seção "Desktop Frontend Architecture" do [AGENTS.md](../../AGENTS.md).
