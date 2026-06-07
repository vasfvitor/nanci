# Plano de Implementacao: Stronghold de Senhas de Certificados

## Objetivo

Permitir que o usuario digite a senha de certificados PFX/P12 apenas quando necessario, sem persistir a senha em texto limpo no SQLite, em arquivos de configuracao ou em logs.

A senha sera armazenada no cofre nativo do sistema operacional por meio da biblioteca `github.com/zalando/go-keyring`. No Windows, isso usa o Windows Credential Manager.

## Escopo

- Criar um `CredentialProvider` decorador em `internal/app` que tenta recuperar a senha no keyring antes de acionar o provedor interativo existente.
- Reutilizar os provedores atuais:
  - CLI: `internal/cli/credential.go`
  - Desktop/Wails: `internal/desktop/app.go`
- Validar senhas recuperadas ou digitadas com `internal/foundation/cert.LoadPKCS12` antes de usa-las e antes de salva-las.
- Integrar o decorador na criacao da aplicacao CLI e Desktop.
- Adicionar testes unitarios para o fluxo de cache/fallback sem depender do keyring real do sistema operacional.

## Fora de Escopo

- Migrar dados do banco, pois senhas de certificado nao devem existir no schema atual.
- Criar tela de gerenciamento de senhas salvas no Windows Credential Manager.
- Trocar a biblioteca de leitura PKCS#12.
- Implementar frontend novo para "lembrar senha"; o comportamento sera automatico.

## Dependencia

Adicionar:

```sh
go get github.com/zalando/go-keyring
```

Notas de plataforma:

- Windows: usa Windows Credential Manager.
- Linux: usa Secret Service via DBus.
- macOS: usa o utilitario nativo `security`.

Como a aplicacao alvo principal e Windows, a primeira validacao manual deve ser feita no Windows Credential Manager.

## Design

### Novo provider

Criar `internal/app/keyring.go` com um decorador:

```go
type KeyringCredentialProvider struct {
    Next    CredentialProvider
    Service string
    Store   KeyringStore
}
```

`Next` e o provedor interativo usado como fallback. `Service` deve ter o valor padrao `nanci_certs`. `Store` e uma interface interna pequena para permitir testes:

```go
type KeyringStore interface {
    Get(service, user string) (string, error)
    Set(service, user, password string) error
    Delete(service, user string) error
}
```

A implementacao de producao chama `keyring.Get`, `keyring.Set` e `keyring.Delete`.

### Identificador da senha

Usar `CertPasswordRequest.CredentialID` como identificador primario no keyring.

Formato recomendado da chave:

```text
credential:<CredentialID>
```

Motivo: o ID da credencial e estavel mesmo se o usuario renomear o rotulo ou alterar o caminho do certificado. Quando `CredentialID` estiver vazio por alguma razao inesperada, retornar erro claro em vez de salvar com chave ambigua.

### Fluxo de leitura

Quando `GetCertPassword(ctx, req)` for chamado:

1. Validar que `Next` esta configurado.
2. Validar que `req.CredentialID` e `req.CertPath` nao estao vazios.
3. Tentar ler `credential:<CredentialID>` do keyring.
4. Se encontrar senha:
   - validar com `cert.LoadPKCS12(req.CertPath, senha)`;
   - se valida, retornar a senha sem abrir prompt;
   - se a senha for invalida, apagar a entrada antiga do keyring e seguir para fallback interativo;
   - se o certificado nao existir ou estiver invalido por outro motivo, retornar o erro, pois pedir a senha novamente nao resolve esses casos.
5. Se nao encontrar senha, ou se o keyring estiver indisponivel, chamar `Next.GetCertPassword(ctx, req)`.
6. Validar a senha digitada com `cert.LoadPKCS12`.
7. Se valida, salvar no keyring com `Set`.
8. Retornar a senha.

### Tratamento de erros

- Senha ausente no keyring: seguir para o provider interativo.
- Keyring indisponivel ou erro de permissao ao ler: registrar em debug/warn se houver logger disponivel e seguir para o provider interativo.
- Senha salva invalida: apagar a entrada e pedir novamente.
- Erro ao salvar no keyring depois de senha valida: nao bloquear a sincronizacao; retornar a senha valida e registrar o erro.
- Cancelamento do usuario no provider interativo: propagar o erro.
- `cert.ErrInvalidPass` depois do prompt: retornar erro claro de senha invalida e nao salvar.

## Arquivos Afetados

### `go.mod` / `go.sum`

- Adicionar `github.com/zalando/go-keyring`.

### `internal/app/keyring.go` (novo)

- Implementar `KeyringCredentialProvider`.
- Implementar `KeyringStore` e adaptador de producao para `github.com/zalando/go-keyring`.
- Centralizar constantes:
  - service padrao: `nanci_certs`
  - prefixo de chave: `credential:`
- Validar senha com `cert.LoadPKCS12`.

### `internal/app/keyring_test.go` (novo)

Cobrir os principais fluxos com fakes:

- retorna senha do keyring quando ela existe e e valida;
- chama fallback quando a senha nao existe;
- salva no keyring depois de senha digitada valida;
- apaga senha antiga quando a senha salva e invalida;
- nao salva senha digitada invalida;
- nao falha a sincronizacao quando `Set` falha depois de uma senha valida.

Para evitar depender de certificados reais nos testes do provider, considerar injetar uma funcao de validacao:

```go
Validate func(certPath, password string) error
```

Em producao, essa funcao chama `cert.LoadPKCS12` e descarta o retorno.

### `internal/cli/root.go`

Alterar a injecao atual:

```go
CredentialProvider: TerminalCredentialProvider{In: os.Stdin, Out: os.Stderr},
```

para um wrapper:

```go
CredentialProvider: app.NewKeyringCredentialProvider(
    TerminalCredentialProvider{In: os.Stdin, Out: os.Stderr},
),
```

O nome exato do construtor pode variar, mas a chamada deve manter o CLI sem dependencia direta de `go-keyring`.

### `internal/desktop/app.go`

Envolver o `WailsCredentialProvider` com o mesmo provider:

```go
CredentialProvider: app.NewKeyringCredentialProvider(WailsCredentialProvider{
    ctx:           ctx,
    passwordChans: a.passwordChans,
    mu:            &a.mu,
}),
```

Assim o dialogo de senha so aparece quando o cofre nao tiver uma senha valida.

## Consideracoes de Seguranca

- Nunca logar senha, nem tamanho da senha.
- Nunca persistir senha no banco ou em arquivo de configuracao.
- Usar apenas `CredentialID` na chave do keyring; nao incluir CNPJ, caminho local completo ou rotulo caso isso exponha informacao sensivel desnecessaria.
- Apagar entradas antigas quando a senha salva falhar com `cert.ErrInvalidPass`.
- Manter o fallback interativo para ambientes onde o keyring nao esteja disponivel.

## Consideracoes de Performance

O provider validara a senha com `cert.LoadPKCS12` antes de retorna-la. Isso significa que o certificado sera carregado uma vez no provider e outra vez em `App.Pull`, que hoje tambem chama `cert.LoadPKCS12`.

Esse custo e aceitavel para a primeira versao porque evita salvar senha incorreta no keyring. Uma otimizacao futura pode alterar o contrato para retornar o certificado ja carregado, mas isso aumentaria o escopo e tocaria mais camadas.

## Plano de Implementacao

1. Adicionar `github.com/zalando/go-keyring` ao modulo.
2. Criar `internal/app/keyring.go` com o decorador e o adaptador de keyring.
3. Adicionar testes unitarios em `internal/app/keyring_test.go`.
4. Envolver o provider do CLI em `internal/cli/root.go`.
5. Envolver o provider do Desktop em `internal/desktop/app.go`.
6. Rodar `gofmt`/`goimports`.
7. Rodar `go test ./...`.
8. Fazer validacao manual no Windows.

## Plano de Verificacao

### Automatizado

```sh
go test ./...
```

Se houver alteracoes de frontend ou Wails gerado, rodar tambem:

```sh
pnpm run lint
```

em `internal/desktop/frontend`.

### Manual no Windows

1. Remover qualquer entrada antiga relacionada ao servico `nanci_certs` no Windows Credential Manager.
2. Rodar o Nanci e executar uma sincronizacao.
3. Confirmar que a senha e solicitada na primeira execucao.
4. Confirmar que a sincronizacao conclui e que uma credencial foi criada no Windows Credential Manager.
5. Fechar e abrir o Nanci novamente.
6. Executar a mesma sincronizacao.
7. Confirmar que a senha nao e solicitada.
8. Alterar ou remover a senha no Windows Credential Manager.
9. Executar a sincronizacao novamente.
10. Confirmar que o Nanci volta a pedir a senha e atualiza o keyring apos uma senha valida.

## Riscos e Mitigacoes

- Keyring indisponivel em ambientes Linux/headless: manter fallback interativo e nao bloquear o uso.
- Entrada salva com senha incorreta: validar antes de salvar e apagar entradas invalidas.
- Mudanca de arquivo de certificado na mesma credencial: validar a senha salva contra o `CertPath` atual; se falhar por senha invalida, pedir novamente.
- Duplicacao temporaria de carregamento do certificado: aceitar nesta versao para manter a mudanca pequena e segura.
