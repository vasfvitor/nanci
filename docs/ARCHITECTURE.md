# Arquitetura do Nanci

## Estrutura de Diretórios

- `cmd/nanci`: Ponto de entrada (main) do CLI.
- `internal/cli`: Adaptadores e configuração dos comandos Cobra (ex: `nanci pull`, `nanci export`, `nanci nfe`).
- `internal/desktop`: O aplicativo Wails completo. Contém o código Go que faz ponte com o frontend, e dentro dele, `frontend/` com o código Vue 3 / TypeScript.
- `internal/app`: Casos de uso centrais. Responsável por inicializar as dependências e amarrar repositórios com serviços. Os casos de uso de NF-e ficam em `NFeService` (`internal/app/nfe*.go`).
- `internal/sync`: Loop de sincronização por NSU, genérico por origem (NFS-e e NF-e), com orçamento de consultas, bloqueios e carregamento do certificado (`CertificateLoader`).
- `internal/store`: Camada de persistência. Contém as queries (frequentemente geradas via sqlc), conexões SQLite e a pasta `migrations_v2/` com o schema do banco.
- `internal/nfse` e `internal/adn`: Modelos da Nota Fiscal de Serviço Eletrônica e cliente da API do Ambiente de Dados Nacional.
- `internal/dfe`: Vocabulário comum dos documentos fiscais eletrônicos da SEFAZ (NF-e, CT-e): chave de acesso de 44 caracteres, valores monetários (`Money`), `CompanyID` e `ErrInvalidEnum`. Sem código de rede ou banco.
- `internal/nfe`: Modelos da NF-e (modelo 55): leitura de `resNFe`, `procNFe` e eventos, papel da empresa na nota, regras de mesclagem resumo/completa, estado e prazos da manifestação.
- `internal/sefaz`: Cliente SOAP dos webservices do Ambiente Nacional da NF-e (`NFeDistribuicaoDFe` e `NFeRecepcaoEvento4`) e assinatura XMLDSig dos eventos. Só fala o protocolo; regras de armazenamento e sincronização ficam com quem chama.
- `internal/report`: Classes de exportação que formatam os dados do banco para `.xlsx`, `.csv` e `.zip`.
- `internal/foundation`: Utilitários gerais do projeto (parsers de CNPJ, códigos de UF em `uf`, handlers de build, criptografia).
- `internal/foundation/httpclient`: Cliente HTTP mTLS neutro compartilhado por `internal/adn` e `internal/sefaz`: transporte TLS, retry, leitura limitada do corpo e registros de log. Quem chama define cabeçalhos, formato do payload, o significado de cada status e como mascarar identificadores (CNPJ, chave de acesso) antes de irem para o log.
- `internal/foundation/redact`: Mascaramento de identificadores fiscais (CNPJ, CPF, chave de acesso, nomes) em XML de NF-e, CT-e e NFS-e, usado nos logs de `internal/adn`, `internal/sefaz` e `internal/sync`.
- `internal/foundation/xmlwalk`: Leitura de XML em fluxo que entrega cada elemento pelo caminho a partir da raiz (`Walk`, `HasAnySuffix`, `AttrValue`), para os parsers de documentos fiscais casarem campos por sufixo de caminho. Usado por `internal/nfe`.
- `third_party/gonfe`: Licença do [gonfe](https://github.com/mschunke/gonfe) (MIT), do qual partes de `internal/sefaz` foram adaptadas.

## Fluxo de Dados

1. **Entrada de Dados**: O usuário interage pelo Frontend Wails ou pelo Terminal CLI.
2. **Orquestração**: A requisição bate no `internal/app` que delega para o `internal/sync`.
3. **Comunicação Segura**: O `sync` carrega o certificado da empresa e consulta a origem escolhida por mTLS: `internal/adn` para NFS-e, `internal/sefaz` para NF-e.
4. **Persistência Bruta**: Os XMLs recebidos são validados no `internal/nfse` ou `internal/nfe` e salvos de forma intacta no armazenamento de blobs, com os metadados no SQLite via `internal/store`.
5. **Processamento**: Partes vitais do XML (Emitente, Tomador/Destinatário, Valor, Descrição) são cacheadas em colunas específicas no banco para pesquisa rápida e exportação.

## Origens de Sincronização

Cada serviço de distribuição por NSU é uma implementação da interface `Source` em `internal/sync/source.go`:

- `Kind()` identifica a origem (`nfse`, `nfe`; `cte` já é aceito pelo schema).
- `Policy()` informa o intervalo entre requisições e o limite de consultas por hora (0 = sem limite). A NFS-e não tem limite; a NF-e tem 20 por hora.
- `Fetch()` busca a página seguinte ao cursor e traduz a resposta em um `Batch`: itens, próximo cursor, se deve parar e até quando esperar (`WaitUntil`).
- `ProcessItem()` decodifica e grava um item e chama `commit` exatamente uma vez, para que o checkpoint avance na mesma transação.

O loop (`internal/sync/loop.go`) é o mesmo para todas as origens. Ele desconta o orçamento antes de cada requisição, grava o bloqueio pedido pela origem e pula, como não suportado, um item que falha ao ser interpretado três vezes seguidas no mesmo NSU. O `Manager` recusa o pull de uma origem bloqueada antes de pedir a senha, e não deixa dois pulls da mesma empresa e origem rodarem ao mesmo tempo.

O estado é separado por empresa e origem:

- `sync_state` e `sync_runs`: cursor, checkpoint e histórico de execuções, com a coluna `source`.
- `company_sync_sources`: data em que a sincronização inicial terminou e `blocked_until`/`blocked_reason`. Zerar o estado local limpa a sincronização inicial, mas mantém o bloqueio.
- `sync_requests`: uma linha por requisição enviada, para a janela móvel de 1 hora do orçamento.

Uma nova origem (por exemplo, CT-e) precisa de um `Source`, tabelas e tela próprias; o loop, o orçamento, o `httpclient` e o `CertificateLoader` servem sem mudança. Os detalhes da NF-e estão em [NFE_SEFAZ.md](NFE_SEFAZ.md).

## Regras de Dependência

O `golangci-lint` (regra `depguard` em `.golangci.yml`) impede que:

- `internal/app` dependa de `internal/store` ou de `database/sql`;
- os pacotes de domínio `internal/nfse`, `internal/nfe` e `internal/dfe` dependam de `internal/app`, `internal/store` ou `database/sql`.
