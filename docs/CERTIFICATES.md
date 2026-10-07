# Certificados Digitais A1

Como o Nanci carrega o certificado, guarda a senha e o usa com o ADN e a SEFAZ. O guia de uso está no site: [Certificado A1](https://vasfvitor.github.io/nanci/docs/certificados/).

## Carregamento

- Formatos `.pfx` e `.p12` (PKCS#12). A leitura usa o fork em `third_party/go-pkcs12`, que aceita a codificação BER de certificados emitidos por algumas ACs.
- O arquivo não é copiado. A tabela `credentials` guarda o caminho (`cert_path`), conferido com `os.Stat` antes do uso (`internal/credential`). `credential update-path` e a tela Credenciais trocam o caminho.
- Uma credencial pode servir a várias empresas (`company assign-credential`).

## Senha

A senha nunca vai para o SQLite. A ordem de busca (`internal/app/keyring.go`):

1. Cofre do sistema operacional, via `github.com/zalando/go-keyring`, serviço `nanci_certs`, chave = ID da credencial. A senha guardada é testada contra o arquivo; se não abrir mais o certificado, segue para o passo 2.
2. Fallback do chamador: no CLI, `NANCI_CERT_PASSWORD` e depois o prompt no terminal (`internal/cli/credential.go`); no desktop, o diálogo de senha (`PasswordPromptDialog`). O desktop não lê `NANCI_CERT_PASSWORD`.
3. Uma senha que abre o certificado é gravada no cofre.

O pedido de senha leva a finalidade ("Sincronização NF-e", "Assinatura: Ciência da Operação (12 notas)"), mostrada no diálogo do desktop. A senha é pedida uma vez por operação: na Ciência da Operação em lote, uma senha assina todos os lotes.

### `.env.local`

`internal/foundation/envfile` carrega `.env.local` da pasta atual, da pasta do executável e da pasta de dados (`%LOCALAPPDATA%\nanci`, ou `os.UserConfigDir()/nanci` fora do Windows), nessa ordem. Variáveis já definidas no processo prevalecem.

## Uso com a SEFAZ (NF-e e CT-e)

O mesmo certificado serve à NFS-e, à distribuição de NF-e e CT-e e à Manifestação do Destinatário. Hosts do Ambiente Nacional: `www1.nfe.fazenda.gov.br`, `www.nfe.fazenda.gov.br` e `hom1.nfe.fazenda.gov.br` (NF-e); `www1.cte.fazenda.gov.br` e `hom1.cte.fazenda.gov.br` (CT-e).

- **Renegociação TLS**: os servidores da SEFAZ só pedem o certificado do cliente renegociando a conexão depois do primeiro handshake. O transporte comum (`internal/foundation/httpclient`) permite renegociação, fica em TLS 1.2 e HTTP/1.1.
- **Cadeias públicas**: os certificados dos servidores são emitidos por cadeias públicas já confiáveis no sistema. O Nanci não embute raízes ICP-Brasil e nunca desliga a verificação do servidor.
- **Raiz do CNPJ**: a empresa precisa ter a mesma raiz (8 primeiros dígitos) do certificado. O Nanci confere antes de enviar; a SEFAZ recusa a consulta (`cStat` 593, [NT 2014.002](https://www.nfe.fazenda.gov.br/portal/exibirArquivo.aspx?conteudo=uWO2d/gTuWg=) v1.40, regra H04) ou o evento (`cStat` 213, [NT 2020.001](https://www.nfe.fazenda.gov.br/portal/exibirArquivo.aspx?conteudo=%2BWTd7iuD21s%3D) v1.60, §6.3.6) quando a raiz difere. O certificado da matriz serve para as filiais.
- **Assinatura de eventos**: as manifestações são assinadas com a chave privada do certificado, que precisa ser RSA (como nos A1 ICP-Brasil). Detalhes em [NFE_SEFAZ.md](NFE_SEFAZ.md#assinatura).

`nanci nfe testar-conexao` e `nanci cte testar-conexao` carregam o certificado e fazem só o handshake TLS com o host de distribuição, sem consumir consultas. Veja [NFE_SEFAZ.md](NFE_SEFAZ.md) e [CTE_SEFAZ.md](CTE_SEFAZ.md).

## Testes

O certificado mock fica em `internal/foundation/cert/testdata/` (senha `mockdata`, CNPJ `70860312000150`) e é recriado com `make mockcert`. Os testes de compatibilidade PKCS#12/BER podem validar também um certificado real mantido fora do repositório.

> [!CAUTION]
> Nunca adicione certificados `.pfx`/`.p12` ou senhas ao repositório ou a issues. O `.gitignore` ignora `*.pfx` e `*.p12` (exceto o mock), mas confira antes de commitar.
