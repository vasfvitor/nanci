# Arquitetura do Nanci

## Estrutura de Diretórios

O CLI (`cmd/nanci`, comandos em `internal/cli`) e o aplicativo (`internal/desktop`, módulo Go separado com o frontend Vue em `frontend/`) usam o mesmo núcleo: casos de uso em `internal/app`, sincronização em `internal/sync` e banco em `internal/store`. Os pacotes cujo papel não sai do nome:

- `internal/syncstate`: estado da sincronização por empresa e origem, sem o loop (`SyncSource`, `SyncRun`, `SyncState`, os enums e o evento de progresso). Pacote folha, só importa `internal/dfe`. Não fica em `internal/sync` porque `sync` importa `store` e `store` precisa desses tipos.
- `internal/dfe`: vocabulário comum da NF-e e do CT-e (chave de 44 caracteres, `Money`) e o que todas as origens usam, NFS-e inclusive (`CompanyID`, `CredentialID`, `Environment`). Sem código de rede ou banco.
- `internal/company`: a empresa precisa ter a mesma raiz de CNPJ da credencial.
- `internal/credential`: certificados A1 reutilizáveis entre empresas. A senha não fica aqui, e sim no cofre do sistema, a cargo de `internal/app` (ver [CERTIFICATES.md](CERTIFICATES.md)).
- `internal/store/sqlgen`: gerado pelo sqlc a partir de `internal/store/queries`, com o schema lido das migrations goose em `migrations_v2/`. Não editar à mão.
- `internal/files`: os XMLs brutos ficam em `blobs/`, um arquivo por hash SHA-256.
- `internal/danfse`: o DANFSe é gerado localmente (`go-danfse-v2`), sem chamar a API do ADN.
- `internal/danfe` e `internal/dacte`: DANFE e DACTE pela biblioteca [fiscal-renderer](https://github.com/awafinance/fiscal-renderer) (LGPL-3.0). O DACTE só tem o leiaute do CT-e modelo 57: CT-e OS, GTV-e e CT-e Simplificado ficam sem PDF (`cte.Document.HasDACTE`).
- `internal/sefaz`: só fala o protocolo SOAP da SEFAZ e assina os eventos; armazenamento e sincronização ficam com quem chama.
- `internal/report`: só os escritores (CSV, XLSX, ZIP de XML e de PDF) são compartilhados; cada origem monta as próprias linhas e entradas.
- `internal/foundation/httpclient`: cliente HTTP mTLS neutro de `internal/adn` e `internal/sefaz` (TLS, retry, leitura limitada, log). Quem chama define cabeçalhos, payload, o significado de cada status e como mascarar identificadores antes do log.
- `internal/foundation/redact`: `internal/sefaz` e `internal/adn` mascaram CNPJ, CPF, chaves e nomes nos corpos que vão para o log e para as mensagens de erro; `internal/adn` também nas URLs. Nas respostas JSON do ADN a máscara não alcança o XML, que vem em base64 com gzip (`ArquivoXml`).
- `internal/foundation/xmlwalk`: leitura de XML em fluxo por caminho, para os parsers da NF-e e do CT-e casarem campos por sufixo.
- `third_party/gonfe`: licença do [gonfe](https://github.com/mschunke/gonfe) (MIT), do qual partes de `internal/sefaz` foram adaptadas.

## Fluxo de Dados

O XML recebido é guardado intacto em `blobs/`. Os campos usados em pesquisa e exportação (emitente, tomador ou destinatário, valor, descrição) são copiados para colunas do SQLite.

## Origens de Sincronização

Cada serviço de distribuição por NSU é uma implementação da interface `Source` em `internal/sync/source.go`. `ProcessItem()` chama `commit` exatamente uma vez, para que o checkpoint avance na mesma transação. O limite de consultas por hora de cada origem está em [NFE_SEFAZ.md](NFE_SEFAZ.md#limites-de-consulta); a NFS-e não tem limite.

O loop (`internal/sync/loop.go`) é o mesmo para todas as origens. Ele desconta o orçamento antes de cada requisição, grava o bloqueio pedido pela origem e pula, como não suportado, um item que falha ao ser interpretado três vezes seguidas no mesmo NSU. O `Manager` recusa o pull de uma origem bloqueada antes de pedir a senha, e não deixa duas operações da mesma empresa e origem (pull, busca pela chave, recuperação de NSU, redefinição) rodarem ao mesmo tempo: `ReserveSource` marca a origem em memória, contra o mesmo processo, e em `source_locks`, contra outro processo do nanci (o aplicativo e a linha de comando usam o mesmo banco). A reserva no banco vale 1 minuto e é renovada enquanto a operação roda, então a de um processo que caiu vence sozinha.

O estado é separado por empresa, origem e ambiente. Zerar o estado local limpa a sincronização inicial em todos os ambientes, mas mantém o bloqueio (`company_sync_sources.blocked_until`).

Uma nova origem precisa de um `Source`, tabelas e tela próprias; o loop, o orçamento, o `httpclient` e o `CertificateLoader` servem sem mudança, como aconteceu com o CT-e. Os detalhes da NF-e estão em [NFE_SEFAZ.md](NFE_SEFAZ.md) e os do CT-e em [CTE_SEFAZ.md](CTE_SEFAZ.md).

## Regras de Dependência

`internal/app` usa `internal/store` e `internal/sync` direto, de propósito: a camada de casos de uso é fina e os repositórios são testados com SQLite real, então uma interface entre os dois só repetiria as assinaturas.

Os pacotes de domínio `internal/nfse`, `internal/nfe`, `internal/cte`, `internal/dfe` e `internal/syncstate` não importam `internal/app`, `internal/store` nem `database/sql`. A regra `domain` do `depguard` em `.golangci.yml` cobra isso também no Windows (conferido em 20/11/2026 com um import de `database/sql` em `internal/dfe`).
