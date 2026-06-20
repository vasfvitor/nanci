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

## Cuidados Importantes de Segurança

> [!CAUTION]
> **NUNCA** adicione seus arquivos de certificado digital `.pfx` / `.p12` ou suas respectivas senhas no repositório Git ou em issues públicas. 
> 
> O Nanci processa as credenciais de forma local-first para garantir que as chaves privadas nunca transitem por servidores de terceiros.
