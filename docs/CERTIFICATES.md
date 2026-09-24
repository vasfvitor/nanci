# Certificados Digitais A1

Como gerenciar senhas e o uso de certificados digitais e-CNPJ no formato A1 no Nanci.

---

O Nanci suporta certificados digitais e-CNPJ tipo A1 nos formatos `.pfx` e `.p12`.

## Gerenciamento da Senha do Certificado

Por padrão, a senha do certificado é solicitada via interface gráfica no momento do uso (ao cadastrar a credencial ou ao sincronizar caso não tenha sido salva localmente).

Para uso automatizado via linha de comando (CLI), a senha pode ser injetada via variável de ambiente:

```bash
export NANCI_CERT_PASSWORD="sua-senha-aqui"
```

## Arquivo de Configuração `.env.local`

Para desenvolvimento ou uso avançado em automações recorrentes, o Nanci carrega variáveis de ambiente de um arquivo `.env.local`. O app busca por este arquivo nos seguintes caminhos, por ordem de prioridade:

1. No diretório de trabalho atual (onde o comando foi invocado).
2. No diretório onde o executável do Nanci está localizado.
3. Na pasta de dados do usuário: `%LOCALAPPDATA%\nanci\.env.local` (Windows) ou `~/.nanci/.env.local` (Linux/macOS).

**Exemplo de conteúdo para o `.env.local`:**

```dotenv
NANCI_CERT_PASSWORD=senha-super-secreta
```

## Uso do Certificado com a SEFAZ (NF-e e CT-e)

O mesmo certificado A1 cadastrado para a NFS-e é usado na distribuição de NF-e, na Manifestação do Destinatário e na distribuição de CT-e. Não é preciso outro certificado nem outra configuração. Os hosts do Ambiente Nacional são `www1.nfe.fazenda.gov.br`, `www.nfe.fazenda.gov.br` e `hom1.nfe.fazenda.gov.br` para a NF-e, e `www1.cte.fazenda.gov.br` (produção) e `hom1.cte.fazenda.gov.br` (homologação) para o CT-e. Pontos específicos do Ambiente Nacional da SEFAZ:

- **Renegociação TLS**: os servidores da SEFAZ só pedem o certificado do cliente depois do primeiro handshake, renegociando a conexão. O Nanci permite essa renegociação (TLS 1.2, HTTP/1.1); nada precisa ser configurado.
- **Cadeias públicas**: os certificados dos servidores da SEFAZ são emitidos por autoridades públicas já confiáveis no sistema operacional. Não é preciso instalar raízes ICP-Brasil, e a verificação do servidor nunca é desligada.
- **Raiz do CNPJ**: a empresa consultada precisa ter a mesma raiz de CNPJ (8 primeiros caracteres) do certificado. O Nanci confere isso antes de qualquer envio; a SEFAZ também recusa a consulta (`cStat` 593) ou o evento (`cStat` 631) quando a raiz difere. O certificado da matriz serve para as filiais.
- **Assinatura de eventos**: as manifestações são assinadas com a chave privada do certificado, que precisa ser RSA (como nos A1 ICP-Brasil).
- **Senha por operação**: a senha é pedida uma vez por operação. Na Ciência da Operação em lote, uma única senha assina todos os lotes. O pedido informa a finalidade, por exemplo "Sincronização NF-e" ou "Assinatura: Ciência da Operação (12 notas)"; no desktop ela aparece no diálogo de senha. `NANCI_CERT_PASSWORD` também vale para esses comandos.

Os comandos `nanci nfe testar-conexao --cnpj <CNPJ>` e `nanci cte testar-conexao --cnpj <CNPJ>` carregam o certificado e testam o TLS com o host de distribuição de cada serviço sem consumir consultas. Veja [NFE_SEFAZ.md](NFE_SEFAZ.md) e [CTE_SEFAZ.md](CTE_SEFAZ.md).

## Cuidados Importantes de Segurança

> [!CAUTION]
> **NUNCA** adicione seus arquivos de certificado digital `.pfx` / `.p12` ou suas respectivas senhas no repositório Git ou em issues públicas. 
> 
> O Nanci processa as credenciais de forma local-first para garantir que as chaves privadas nunca transitem por servidores de terceiros.
