# Nanci CLI (NFS-e Sync)

## Funcionalidades

- **Sincronização Incremental**: Busca as Notas Fiscais de Serviço (NFS-e) por NSU de forma incremental, com checkpoint local por empresa.
- **Leitura Nativa de Certificados**: Autenticação mTLS lendo diretamente arquivos `.pfx` ou `.p12` de certificados A1, sem depender de ferramentas do SO.
- **Persistência Local (SQLite)**: Banco de dados auto-contido para gerenciar contribuintes, reter histórico e indexar os documentos baixados.
- **Modelo Canônico + Visão por Empresa**: Um mesmo `chave_acesso` pode ser associado a múltiplas empresas gerenciadas, preservando `company_role` e `visibility_reason` por empresa.
- **Parser Avançado**: Descompacta o payload do governo (Base64 + GZip), extrai os dados essenciais e calcula o hash SHA-256 para integridade.
- **Exportação Rica**:
  - Geração de planilhas Excel prontas para uso contábil (`.xlsx`) com formatação automática de moeda.
  - Geração de tabelas CSV (`.csv`) para compatibilidade com ERPs e importadores legados.
  - Exportação em lote de arquivos físicos em `.zip`.
- **Suporte Inicial a Eventos**: O parser detecta eventos distribuídos e já persiste atualizações básicas de status, com o redesenho completo do ciclo de eventos ainda em andamento.

---

## Como Instalar

Certifique-se de que o **Go 1.21+** está instalado.

```bash
git clone https://github.com/vasfvitor/nanci.git
cd nanci
go build -o nanci.exe ./cmd/nanci
```

---

## Como Usar (Guia Rápido)

### 1. Inicializar o Ambiente
Cria os diretórios e o banco de dados na sua pasta de usuário (`~/.nanci` ou `%APPDATA%\nanci`).
```bash
./nanci.exe init
```

### 2. Adicionar uma Empresa (Contribuinte)
Cadastra uma empresa no banco ligando-a ao seu respectivo certificado digital A1.
```bash
./nanci.exe company add --cnpj 12345678000199 --name "Minha Empresa" --cert "C:\Caminho\para\certificado.pfx"
```
Você pode definir a variável de ambiente `NANCI_CERT_PASSWORD=sua-senha` para não precisar digitá-la interativamente nas execuções.

### 3. Sincronizar (Pull)
Conecta à Receita Federal e baixa todos os documentos novos disponíveis.
```bash
./nanci.exe pull --cnpj 12345678000199
```

### 4. Listar
Visualiza um resumo rápido no terminal das notas processadas.
```bash
./nanci.exe list --cnpj 12345678000199 --competencia 2026-06
```

### 5. Exportar
Extraia os dados sincronizados em formatos portáteis.
```bash
# Para gerar uma planilha em Excel
./nanci.exe export xlsx --cnpj 12345678000199 --out "relatorio.xlsx"

# Para gerar um CSV
./nanci.exe export csv --cnpj 12345678000199 --out "relatorio.csv"

# Para exportar todos os arquivos XML baixados em um ZIP
./nanci.exe export zip --cnpj 12345678000199 --out "notas_fiscais.zip"
```

---

## Estrutura do Projeto 

- `cmd/nanci`: Ponto de entrada do executável.
- `internal/cli/`: Interface de Linha de Comando (Cobra) — gerencia flags, validações de entrada e formatação de saídas no terminal.
- `internal/app/`: Lógica de aplicação (Use Cases) — coordena as operações principais de forma agnóstica à interface (CLI, Web, Desktop).
- `internal/service/sync/`: O coração da sincronização de notas — orquestra chamadas à API, salvamento no banco e gravação de arquivos em disco.
- `internal/nfse/`: Entidades de domínio (modelos de empresa e documento fiscal) e regras de negócio centrais (ex: parser XML).
- `internal/adn/`: Client HTTP especializado configurado com mTLS para consumo da API ADN da Receita Federal.
- `internal/store/`: Camada de persistência (SQLite) e gerenciamento de migrações estruturais (`goose`).
- `internal/report/`: Construtores de exportação (planilhas `.xlsx`, relatórios `.csv` e arquivos compactados `.zip`).
- `internal/files/`: Taxonomia e gravação segura de XMLs e payloads binários em disco.
- `internal/foundation/`: Código base genérico (certificados digitais, manipulação de strings, logs, CNPJ validation).

---

## Modelo de Dados Atual

O armazenamento local agora separa:

- `documents`: documento canônico por `chave_acesso`
- `company_documents`: participação da empresa naquele documento, com papel fiscal e motivo de visibilidade

Isso evita conflito quando a mesma NFS-e é relevante para mais de uma empresa cadastrada localmente.

Mais detalhes:

- [docs/storage-model-and-reset-policy.md](docs/storage-model-and-reset-policy.md)

## Política Atual de Migração

O refactor de identidade de documentos foi implementado com política de reset local nesta fase.

Em termos práticos:

- instalações novas funcionam normalmente
- bancos SQLite criados antes da separação entre `documents` e `company_documents` não têm migração in-place ainda
- se você estiver com um banco antigo desta fase anterior, descarte o arquivo local e deixe a aplicação recriá-lo

---

## Desenvolvimento e Qualidade de Código

- **`make fmt`**: Formata o código fonte localmente (`gofmt`) e organiza os imports (`goimports`).
- **`make lint`**: Roda o `golangci-lint` (versão 2) verificando boas práticas, performance, e uso correto de recursos.
- **`make vuln`**: Verifica vulnerabilidades na linguagem e nas dependências utilizando o scanner oficial da linguagem (`govulncheck`).
- **`make security`**: Executa verificações de segurança (`gosec` para vulnerabilidades lógicas e `gitleaks` para detectar senhas/chaves vazadas no código).
- **`make test`**: Roda a suíte completa de testes unitários.
- **`make check`**: Atalho para rodar todos os passos juntos: `fmt`, `vuln`, `lint`, `test` e `security`. Recomendado rodar antes de todo commit.
  
