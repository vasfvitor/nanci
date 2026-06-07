# Plano de Mock Data e Seed Local

O objetivo deste plano e organizar dados falsos, fixtures e ambiente local de desenvolvimento sem misturar mock com regra de negocio real do Nanci.

A regra principal e separar tres coisas diferentes:

```txt
testdata/        arquivos usados por testes do pacote
internal/testutil/ helpers reutilizaveis de teste
cmd/seeddev/     comando explicito para popular ambiente local
```

Mock data nao deve virar uma arquitetura paralela de desenvolvimento, nem entrar no fluxo normal do app. O binario real (`cmd/nanci`) deve continuar inicializando apenas as dependencias reais.

## Estado Atual do Codebase

O Nanci hoje usa esta organizacao:

```txt
cmd/nanci/                 entrada CLI real
internal/app/              casos de uso e wiring da aplicacao
internal/cli/              adaptadores CLI
internal/desktop/          app Wails e frontend Vue
internal/store/            SQLite, migrations e repositories
internal/nfse/             modelo e parsing fiscal
internal/foundation/cert/  leitura PKCS#12/PFX
internal/foundation/cnpj/  validacao de CNPJ
internal/service/sync/     orquestracao de sincronizacao ADN
internal/files/            blob store local
```

Ja existem fixtures de teste em:

```txt
internal/foundation/cert/testdata/
internal/nfse/testdata/
```

Tambem existe um gerador de certificado mock em:

```txt
gen/mock_cert.go
```

Esse gerador cria:

```txt
internal/foundation/cert/testdata/cert_a1_mock_70860312000150.pfx
```

Senha atual:

```txt
mockdata
```

Esse PFX e autoassinado, serve apenas para testes locais de leitura PKCS#12 e inspecao de metadados, e nao funciona contra ADN/NFS-e real.

## Estrutura Final Proposta

Manter a arquitetura atual do Nanci e adicionar apenas os pontos de suporte necessarios:

```txt
cmd/
  nanci/
    main.go
  mockcert/
    main.go
  seeddev/
    main.go

internal/
  app/
    testutil_test.go          opcional, apenas se os testes de app crescerem

  foundation/
    cert/
      pfx.go
      pfx_test.go
      testdata/
        README.md
        cert_a1_mock_70860312000150.pfx

  nfse/
    testdata/
      *.xml

  store/
    seed/
      seed.go
      companies.go
      credentials.go
      documents.go

  testutil/
    fixtures/
      app.go
      company.go
      credential.go
      document.go
      paths.go

devdata/
  .gitkeep
```

Notas:

- `cmd/mockcert` substitui `gen/mock_cert.go` como ferramenta de repo.
- `cmd/seeddev` popula banco e diretorios locais explicitamente.
- `internal/store/seed` contem a logica de seed porque conhece SQLite, repositorios e caminhos locais.
- `internal/testutil/fixtures` contem builders/factories reutilizaveis por testes.
- `devdata/` fica ignorado pelo git e guarda banco SQLite, XMLs locais e copias de PFX geradas localmente.

## O Que Fica Fora

Nao adicionar mock data em pacotes de regra de negocio:

```txt
internal/app/mockdata.go
internal/nfse/mockdata.go
internal/service/sync/mockdata.go
```

Nao espalhar flags como:

```go
if dev {
    // cria empresa mock
}
```

O ambiente de desenvolvimento deve ser montado no wiring de ferramentas (`cmd/seeddev`), nao dentro da regra de negocio usada em producao.

## Certificado Mock

Mover o gerador atual:

```txt
gen/mock_cert.go
```

Para:

```txt
cmd/mockcert/main.go
```

Uso:

```bash
go run ./cmd/mockcert
```

O comando deve gerar, por padrao:

```txt
internal/foundation/cert/testdata/cert_a1_mock_70860312000150.pfx
```

Manter o certificado mock perto do pacote que consome o arquivo em teste (`internal/foundation/cert`), porque ele e fixture de leitura PKCS#12, nao dado de aplicacao.

Adicionar:

```txt
internal/foundation/cert/testdata/README.md
```

Conteudo sugerido:

```txt
# Certificado A1 Mock

Arquivo: cert_a1_mock_70860312000150.pfx
Senha: mockdata
CNPJ: 70860312000150

Uso: apenas testes locais.
Este certificado e autoassinado.
Nao e um certificado ICP-Brasil real.
Nao funciona contra ADN/NFS-e real.
Nao usar em producao.
```

Como o `.gitignore` atual bloqueia `*.pfx` e `*.p12`, existem duas opcoes:

1. Manter o PFX fora do git e gerar com `go run ./cmd/mockcert`.
2. Versionar apenas este fixture com excecao explicita no `.gitignore`.

Opcao recomendada para testes mais estaveis:

```gitignore
*.pfx
*.p12
!internal/foundation/cert/testdata/cert_a1_mock_70860312000150.pfx
```

Essa excecao deve ser limitada ao fixture mock documentado. Certificados reais continuam proibidos no repositorio.

## Fixtures Para Testes

Criar helpers pequenos em:

```txt
internal/testutil/fixtures/
```

Exemplos de responsabilidades:

```txt
Company()              retorna nfse.Company valida
Credential()           retorna nfse.Credential valida
MockPFXPath(t)         retorna caminho do PFX mock
TempApp(t)             monta App com SQLite temporario
TempStore(t)           abre SQLite temporario com migrations
Document()             retorna nfse.Document valido
```

Esses helpers devem ser deterministas e nao devem acessar rede, keyring ou ADN.

Tambem devem ser seguros para testes com `t.Parallel()`.

Regras para compatibilidade com `t.Parallel()`:

- usar `t.TempDir()` para banco, arquivos e blob store temporarios;
- nao escrever em `devdata/` a partir de testes;
- nao reaproveitar SQLite, diretorios ou arquivos mutaveis entre testes paralelos;
- nao depender de variaveis globais mutaveis;
- retornar copias novas de structs em cada fixture;
- usar IDs estaveis apenas quando o teste precisa testar conflito/upsert;
- preferir IDs unicos por teste quando o dado vai para banco compartilhado, embora o padrao recomendado seja banco temporario por teste.

Exemplo de uso esperado:

```go
func TestAddCompany(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	application := fixtures.TempApp(t)
	credential := fixtures.Credential()

	// test...
	_ = credential
	_ = application
	_ = ctx
}
```

Regra pratica:

- se o dado pertence a um unico pacote, use `testdata/` dentro desse pacote;
- se o helper e reutilizado por varios pacotes, use `internal/testutil/fixtures`;
- se o dado precisa popular um banco local para uso manual, use `cmd/seeddev`.
- se o teste chama `t.Parallel()`, toda fixture mutavel deve estar isolada por `t.TempDir()` ou por instancia propria em memoria.

Para testes dentro do proprio pacote `internal/foundation/cert`, usar caminho relativo ao pacote:

```go
path := filepath.Join("testdata", "cert_a1_mock_70860312000150.pfx")
data, err := os.ReadFile(path)
if err != nil {
	t.Fatal(err)
}
```

Para helpers compartilhados em `internal/testutil/fixtures`, evitar depender do diretorio atual do teste chamador. O helper deve resolver o caminho a partir da raiz do repositorio ou receber o caminho como parametro. Isso evita que um teste em outro pacote quebre por executar com outro working directory.

Quando a API testada recebe caminho, passe `path`. Quando a API testada recebe conteudo binario, leia com `os.ReadFile(path)` e passe `data`. Evitar duplicar fixtures PFX apenas para mudar a forma de entrada do teste.

## Mocks e Fakes em Codigo

Para mocks/fakes em codigo Go, manter o contrato no lado que consome a dependencia e manter a implementacao concreta fora da regra que esta sendo testada.

O codebase atual ja usa esse padrao:

```txt
internal/app/bootstrap.go       define CredentialProvider
internal/nfse/repositories.go   define interfaces de repositorio
internal/files/blobstore.go     define XMLStore
internal/service/sync/sync.go   define documentFetcher privado do pacote
```

Regra pratica:

- interfaces pequenas ficam no pacote consumidor;
- implementacoes reais ficam em `internal/store`, `internal/files`, `internal/adn`, `internal/cli` ou `internal/desktop`;
- fakes usados por um unico pacote ficam no proprio `*_test.go`;
- fakes reutilizados por varios pacotes podem ir para `internal/testutil/fixtures` ou `internal/testutil/fakes`;
- nao criar pacote de mock paralelo para a aplicacao real;
- nao colocar fake em arquivo compilado para producao se ele so existe para teste.

Exemplo no estilo do projeto:

```go
package company

import "context"

type Store interface {
	Create(ctx context.Context, c Company) error
	QueryByCNPJ(ctx context.Context, cnpj string) (Company, error)
}
```

No Nanci atual, esse mesmo principio aparece como interfaces no pacote consumidor/modelo:

```go
package nfse

import "context"

type CompanyRepository interface {
	CreateCompany(ctx context.Context, c *Company) error
	CompanyByCNPJ(ctx context.Context, cnpj string) (*Company, error)
	ListCompanies(ctx context.Context) ([]Company, error)
	AssignCredential(ctx context.Context, companyID CompanyID, credID CredentialID) error
	UpdateCompany(ctx context.Context, id CompanyID, name string, environment Environment) error
}
```

Se o projeto crescer e for feita uma reorganizacao maior no futuro, o equivalente em uma arquitetura `business/platform` seria:

```txt
internal/business/core/company/store.go       contrato consumido pela regra
internal/platform/database/sqlite/company.go  implementacao real
internal/business/core/company/fake_test.go   fake local de teste
internal/testutil/fakestore/company.go        fake compartilhado, se necessario
```

Essa reorganizacao nao e pre-requisito para o plano atual. No Nanci de hoje, o caminho pragmatico e manter os contratos existentes em `internal/nfse`, `internal/app`, `internal/files` e `internal/service/sync`, adicionando fakes apenas onde os testes realmente precisarem.

Fake local do teste:

```go
type credentialProviderStub struct {
	password string
	err      error
}

func (s credentialProviderStub) GetCertPassword(context.Context, app.CertPasswordRequest) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.password, nil
}
```

Quando o fake precisar de estado mutavel em teste paralelo, criar uma instancia por teste e proteger acesso concorrente se o SUT chamar em goroutines.

```go
func TestPullUsesCredentialProvider(t *testing.T) {
	t.Parallel()

	provider := credentialProviderStub{password: "mockdata"}

	// test...
	_ = provider
}
```

Evitar frameworks de mock por padrao. Para este codebase, structs pequenas em teste sao mais simples, legiveis e suficientes na maioria dos casos.

Tambem evitar criar interfaces cedo demais. Criar uma interface apenas quando existe uma necessidade concreta de trocar implementacao, por exemplo:

- banco SQLite;
- cliente HTTP/ADN;
- filesystem/blob store;
- relogio;
- leitura de certificado;
- prompt/keyring de senha.

## Seed Local

Adicionar um comando separado:

```txt
cmd/seeddev/main.go
```

Uso:

```bash
go run ./cmd/seeddev
```

O comando deve criar um ambiente local em:

```txt
devdata/
```

Exemplo:

```txt
devdata/
  nanci-dev.db
  certs/
    cert_a1_mock_70860312000150.pfx
  xml/
    simple-prestada.xml
    simple-tomada.xml
```

Comportamento esperado:

- criar diretorios se nao existirem;
- abrir SQLite com `store.OpenDB(path, true)`;
- copiar o PFX mock para `devdata/certs/`;
- copiar XMLs de `internal/nfse/testdata/` para `devdata/xml/` quando necessario;
- inserir credenciais, empresas e documentos fake;
- ser idempotente.

O comando nao deve ser subcomando de `cmd/nanci` neste primeiro momento. Isso evita misturar ferramenta de desenvolvimento com o binario real distribuivel.

## Upsert, Nao Insert Puro

O seed local deve poder rodar varias vezes sem quebrar por constraint unica.

Hoje as queries principais usam `INSERT` puro:

```txt
internal/store/queries/companies.sql
internal/store/queries/credentials.sql
```

Para seed, nao alterar necessariamente o comportamento dos repositories reais. Em vez disso, adicionar queries especificas de seed ou uma camada `internal/store/seed` com SQL proprio usando upsert.

Exemplo conceitual:

```sql
INSERT INTO credentials (
    id, label, cert_path, environment, owner_cnpj, owner_cnpj_root,
    fingerprint_sha256, subject_name, not_before, not_after, inspected_at,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    label = excluded.label,
    cert_path = excluded.cert_path,
    environment = excluded.environment,
    owner_cnpj = excluded.owner_cnpj,
    owner_cnpj_root = excluded.owner_cnpj_root,
    fingerprint_sha256 = excluded.fingerprint_sha256,
    subject_name = excluded.subject_name,
    not_before = excluded.not_before,
    not_after = excluded.not_after,
    inspected_at = excluded.inspected_at,
    updated_at = excluded.updated_at;
```

Para empresas, usar `ON CONFLICT(cnpj)` se o schema garantir unicidade por CNPJ, ou `ON CONFLICT(id)` caso a chave estavel do seed seja o ID.

IDs de seed devem ser constantes, por exemplo:

```txt
dev-credential-70860312000150
dev-company-70860312000150
```

Isso evita duplicacao a cada execucao.

Exemplo de estrutura de seed idempotente:

```go
func SeedDev(ctx context.Context, store Store) error {
	companies := []Company{
		{
			ID:   "dev-company-70860312000150",
			CNPJ: "70860312000150",
			Name: "Empresa Mock Teste",
		},
	}

	for _, c := range companies {
		if err := store.UpsertCompany(ctx, c); err != nil {
			return err
		}
	}

	return nil
}
```

## Dados Recomendados Para Seed

Seed minimo:

```txt
Credencial:
  id: dev-credential-70860312000150
  label: Certificado Mock 70860312000150
  cert_path: devdata/certs/cert_a1_mock_70860312000150.pfx
  environment: producao_restrita
  owner_cnpj: 70860312000150
  owner_cnpj_root: 70860312

Empresa:
  id: dev-company-70860312000150
  cnpj: 70860312000150
  cnpj_root: 70860312
  name: Empresa Mock Teste
  credential_id: dev-credential-70860312000150
  environment: producao_restrita
  last_nsu: 0
```

Documentos:

- usar XMLs existentes em `internal/nfse/testdata`;
- inserir apenas documentos que exercitam listagem, exportacao e filtros;
- manter datas fixas para testes deterministas;
- nao simular retorno real da ADN dentro de `cmd/nanci`.

## Integracao Com Desktop e CLI

O seed deve ser externo ao app real, mas facil de consumir.

Depois de rodar:

```bash
go run ./cmd/seeddev
```

O usuario pode abrir o app apontando para o banco local em `devdata/`, conforme o mecanismo atual de data dir/configuracao.

Se o codebase ainda nao tiver flag/env clara para data dir, o plano nao deve adicionar `dev` dentro da regra de negocio. A alternativa correta e criar wiring local no proprio `cmd/seeddev` e documentar como abrir o banco gerado.

## Etapas de Implementacao

1. Mover `gen/mock_cert.go` para `cmd/mockcert/main.go`.
2. Adicionar `internal/foundation/cert/testdata/README.md`.
3. Decidir se o PFX mock sera versionado com excecao no `.gitignore` ou sempre gerado localmente.
4. Criar `internal/testutil/fixtures` com builders pequenos para `nfse.Company`, `nfse.Credential`, `nfse.Document` e app/store temporarios.
5. Criar `internal/store/seed` com funcoes idempotentes:
   - `SeedDevelopment(ctx, db, options) error`
   - `UpsertCredential(ctx, db, credential) error`
   - `UpsertCompany(ctx, db, company) error`
   - `UpsertDocuments(ctx, db, documents) error`
6. Criar `cmd/seeddev/main.go` para abrir/criar `devdata/nanci-dev.db`, copiar fixtures e chamar `store/seed`.
7. Adicionar `devdata/` ao `.gitignore`, preservando opcionalmente `devdata/.gitkeep` se quiser documentar a pasta.
8. Atualizar README ou documento de desenvolvimento com os comandos:

```bash
go run ./cmd/mockcert
go run ./cmd/seeddev
```

## Plano de Verificacao

Verificacao automatica:

```bash
go test ./...
```

Verificacao manual:

```bash
go run ./cmd/mockcert
go run ./cmd/seeddev
```

Depois, confirmar:

- `devdata/nanci-dev.db` foi criado;
- rodar `go run ./cmd/seeddev` duas vezes nao duplica empresas nem credenciais;
- a credencial aponta para `devdata/certs/cert_a1_mock_70860312000150.pfx`;
- o PFX abre com a senha `mockdata`;
- o app lista a empresa mock quando apontado para o banco dev;
- nenhuma chamada real para ADN e feita pelo seed.

## Decisoes de Seguranca

- Nunca commitar certificados reais.
- Nunca commitar senhas reais.
- Nunca tratar fixture PFX como certificado valido para producao restrita.
- Documentar claramente todo PFX mock versionado.
- Manter `*.pfx` e `*.p12` ignorados por padrao.
- Limitar excecoes de `.gitignore` a arquivos mock especificos em `testdata/`.

## Resultado Esperado

Ao final, o Nanci tera:

- fixtures de teste perto dos pacotes que as usam;
- helpers de teste compartilhados sem duplicar boilerplate;
- certificado mock gerado por ferramenta explicita de repo;
- seed local idempotente para desenvolvimento manual;
- regra de negocio sem caminhos especiais de mock/dev;
- menor risco de vazar certificado real ou dado fiscal real no repositorio.
