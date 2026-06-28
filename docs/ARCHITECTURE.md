# Arquitetura do Nanci

## Estrutura de Diretórios

- `cmd/nanci`: Ponto de entrada (main) do CLI.
- `internal/cli`: Adaptadores e configuração dos comandos Cobra (ex: `nanci pull`, `nanci export`).
- `internal/desktop`: O aplicativo Wails completo. Contém o código Go que faz ponte com o frontend, e dentro dele, `frontend/` com o código Vue 3 / TypeScript.
- `internal/app`: Casos de uso centrais. Responsável por inicializar as dependências e amarrar repositórios com serviços.
- `internal/service/sync`: Lógica orquestradora de sincronização (buscar NSU, tratar erros de retry, salvar no DB).
- `internal/store`: Camada de persistência. Contém as queries (frequentemente geradas via sqlc), conexões SQLite e a pasta `migrations_v2/` com o schema do banco.
- `internal/nfse` e `internal/adn`: Camadas de domínio responsáveis por definir os modelos de Nota Fiscal de Serviço Eletrônica e as interações com a API do Ambiente de Dados Nacional.
- `internal/report`: Classes de exportação que formatam os dados do banco para `.xlsx`, `.csv` e `.zip`.
- `internal/foundation`: Utilitários gerais do projeto (parsers de CNPJ, handlers de build, criptografia).

## Fluxo de Dados

1. **Entrada de Dados**: O usuário interage pelo Frontend Wails ou pelo Terminal CLI.
2. **Orquestração**: A requisição bate no `internal/app` que delega para o `internal/service/sync`.
3. **Comunicação Segura**: O `sync` utiliza as credenciais armazenadas para fazer mTLS contra o `internal/adn`.
4. **Persistência Bruta**: Os XMLs recebidos são validados no `internal/nfse` e salvos de forma intacta no SQLite via `internal/store`.
5. **Processamento**: Partes vitais do XML (Emitente, Tomador, Valor, Descrição) são cacheadas em colunas específicas no banco para pesquisa rápida e exportação.
