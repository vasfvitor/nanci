# Nanci CLI (NFS-e Sync)

## Funcionalidades

- **Sincronização Resiliente**: Busca as Notas Fiscais de Serviço (NFS-e) por NSU de forma incremental. Se a conexão cair, ele retoma exatamente de onde parou.
- **Leitura Nativa de Certificados**: Autenticação mTLS lendo diretamente arquivos `.pfx` ou `.p12` de certificados A1, sem depender de ferramentas do SO.
- **Persistência Local (SQLite)**: Banco de dados auto-contido para gerenciar contribuintes, reter histórico e indexar os documentos baixados.
- **Parser Avançado**: Descompacta o payload do governo (Base64 + GZip), extrai os dados essenciais e calcula o hash SHA-256 para integridade.
- **Exportação Rica**:
  - Geração de planilhas Excel prontas para uso contábil (`.xlsx`) com formatação automática de moeda.
  - Geração de tabelas CSV (`.csv`) para compatibilidade com ERPs e importadores legados.
  - Exportação em lote de arquivos físicos em `.zip`.
- **Suporte a Eventos**: O parser detecta eventos (como notas canceladas ou substituições) enviados na mesma requisição e já atualiza o status do documento de volta para o banco.

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

- `cmd/nanci`: Ponto de entrada do CLI.
- `internal/app/`: Lógica da aplicação, injeção de dependência e comandos do Cobra.
- `internal/foundation/`: Código agnóstico à regra de negócios (logs, strings, paths, certificados, backoff retry).
- `internal/nfse/`: Tipos de domínio e lógicas de núcleo empresarial (modelos de empresa e nota fiscal).
- `internal/service/sync/`: O coração de orquestração do "Pull" de notas (coordena API, banco e disco).
- `internal/store/`: Camada de persistência (SQLite) com as `migrations` (goose).
- `internal/adn/`: Client HTTP configurado com mTLS para conversar especificamente com a API ADN do Governo.
- `internal/report/`: Geradores de planilhas e arquivos.
- `internal/files/`: Utilitários para a taxonomia e persistência de arquivos físicos.