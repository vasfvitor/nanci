# Contribuindo

## Estrutura do projeto

* `cmd/nanci`: entrada do CLI.
* `cmd/mockcert`: gera certificado mock para testes.
* `cmd/seeddev`: cria o ambiente local em `devdata/`.
* `internal/app`: casos de uso e montagem das dependências.
* `internal/cli`: comandos do CLI.
* `internal/desktop`: app desktop com Vue 3 e Wails.
* `internal/nfse`, `internal/adn`, `internal/service/sync`: domínio fiscal, integração com ADN e processamento de XML.
* `internal/store`: SQLite, repositórios e migrações.
* `internal/testutil/fixtures`: helpers de teste.

## Ambiente local

Para criar o ambiente local:

```bash
go run ./cmd/seeddev
```

Esse comando:

* cria a pasta `devdata/`;
* cria ou atualiza o banco `devdata/nanci-dev.db`;
* aplica as migrações;
* copia o certificado mock para `devdata/certs/`;
* cadastra empresa e credencial de teste.

O comando é idempotente, então pode ser rodado mais de uma vez.

Em uso normal, o Nanci salva dados em `~/.nanci/`. Para desenvolvimento, use `devdata/` para não misturar dados reais com dados de teste.

## Certificado mock

O certificado mock fica em:

```text
internal/foundation/cert/testdata/
```

Dados:

* senha: `mockdata`
* CNPJ: `70860312000150`

Esse certificado serve apenas para testes locais. Ele não tem validade fiscal e não funciona em serviços reais. Só é usado para testar o fluxo até a primeira chamada de api externa

Para recriar o certificado:

```bash
go run ./cmd/mockcert
```

Requer `openssl` instalado.

## Código

Antes de commitar, formate o código:

```bash
gofmt
goimports -local github.com/vasfvitor/nanci
```

Padrões gerais:

* nomes exportados: `CamelCase`;
* nomes internos: `camelCase`;
* commits: `escopo: descrição`.

Exemplos:

```text
cli: corrige parser de cnpj
store: adiciona migracao de credenciais
desktop: ajusta tela de empresas
```

## Testes

Use `testdata/` para arquivos usados por um pacote específico.

Use `internal/testutil/fixtures` para factories e helpers reutilizáveis.

Quando o teste precisar gravar arquivos ou criar banco, use `t.TempDir()`.

Para rodar tudo com race detector:

```bash
go test -race ./...
```
