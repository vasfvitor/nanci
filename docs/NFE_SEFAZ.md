# Integração NF-e e Ambiente Nacional da SEFAZ

Detalhamento técnico sobre como o Nanci baixa NF-e (modelo 55) pela distribuição DF-e do Ambiente Nacional e registra a Manifestação do Destinatário.

---

## O que é a distribuição de NF-e

A SEFAZ mantém no Ambiente Nacional (AN) o webservice **NFeDistribuicaoDFe**, que entrega a um CNPJ os documentos em que ele aparece. Cada documento recebe um NSU (Número Sequencial Único) próprio daquele CNPJ, e a consulta funciona como uma fila: o Nanci envia o último NSU que já leu (`distNSU/ultNSU`) e recebe até 50 documentos seguintes, com os cursores atualizados (`ultNSU` e `maxNSU`).

- **Resumo (`resNFe`)**: dados básicos da nota (chave, emitente, valor, situação). Não é o documento fiscal.
- **Completa (`procNFe`)**: o XML autorizado da NF-e com o protocolo.
- **Eventos (`resEvento`, `procEventoNFe`)**: cancelamento, carta de correção e manifestações.

A fila guarda os documentos por cerca de 90 dias. Por isso, ao contrário da NFS-e, a política inicial da empresa (`--sync-start-policy`) **não** se aplica à NF-e: o primeiro pull traz o que a SEFAZ ainda tiver.

## O que a empresa recebe, por papel

O Nanci classifica o papel da empresa em cada nota (coluna `PAPEL` do `nfe list`), na ordem abaixo:

| Papel | O que chega |
|---|---|
| `destinatario` | Todas as NF-e autorizadas contra o CNPJ, de qualquer UF. Primeiro chega o resumo; o XML completo só é distribuído depois da Ciência da Operação (ou de uma manifestação conclusiva). Cancelamentos chegam sem necessidade de manifestação. |
| `emitente` | A SEFAZ não distribui ao emitente as notas que ele mesmo emitiu. Uma nota só aparece com esse papel quando chega por outro motivo (por exemplo, o CNPJ também está no `autXML`). |
| `transportador` | NF-e em que o CNPJ é o transportador. |
| `autorizado` | NF-e em que o CNPJ foi informado no grupo `autXML` (pessoas autorizadas a obter o XML). |
| `none` | A empresa só compartilha a raiz do CNPJ com alguma das partes, ou o motivo não foi identificado. |

Eventos de uma chave que a empresa ainda não tem localmente são descartados, exceto os que a própria empresa assinou (suas manifestações), que ficam guardados e são ligados à nota quando ela chegar.

## Endpoints Consumidos

As chamadas são SOAP 1.2 sobre HTTPS com o certificado A1 da empresa.

1. **Produção:**
   - Distribuição: `https://www1.nfe.fazenda.gov.br/NFeDistribuicaoDFe/NFeDistribuicaoDFe.asmx`
   - Eventos: `https://www.nfe.fazenda.gov.br/NFeRecepcaoEvento4/NFeRecepcaoEvento4.asmx`
2. **Homologação:**
   - Distribuição: `https://hom1.nfe.fazenda.gov.br/NFeDistribuicaoDFe/NFeDistribuicaoDFe.asmx`
   - Eventos: `https://hom1.nfe.fazenda.gov.br/NFeRecepcaoEvento4/NFeRecepcaoEvento4.asmx`

O ambiente segue o da empresa: `producao` usa produção (`tpAmb` 1) e `producao_restrita` usa homologação (`tpAmb` 2). Para trocar, use `nanci company update --cnpj <CNPJ> --env producao`. As URLs estão em `internal/sefaz/endpoints.go` e só podem ser substituídas pelo código (`sefaz.ClientConfig.Endpoints`, usado nos testes); não existe flag nem variável de ambiente para isso.

Cada NF-e e cada evento guardam o seu `tpAmb`: o do XML (`ide/tpAmb` na `procNFe`, `infEvento/tpAmb` no `procEventoNFe`) ou, nos resumos (`resNFe` e `resEvento`, que não trazem o campo), o do ambiente consultado. Um XML de um ambiente diferente do consultado é guardado com o `tpAmb` dele e recebe um aviso de leitura. A lista de notas, o `nfe status`, as pendências de manifestação, a ciência em lote e a exportação mostram só as notas do ambiente atual da empresa. Por isso o ambiente pode ser trocado a qualquer momento, também depois da primeira sincronização: as notas do outro ambiente deixam de aparecer e voltam quando a empresa retorna a ele. O cursor de sincronização é separado por ambiente, na NF-e e na NFS-e, e o bloqueio da SEFAZ (`company_sync_sources.blocked_until`), a data da carga inicial e o orçamento por hora (`sync_requests`) também são: um bloqueio recebido em produção não adia o pull em homologação, e as consultas de um ambiente não gastam o orçamento do outro. Um bloqueio em vigor continua valendo quando a empresa volta ao ambiente em que foi recebido.

Para apagar as NF-e da empresa, use `nanci nfe reset --cnpj <CNPJ>` (sem `--confirmar` só mostra o que seria removido) ou o botão "Redefinir NF-e" do aplicativo. A redefinição remove as notas da empresa nos dois ambientes, seus eventos e marcas de exportação, e volta o cursor ao NSU 0. Notas que outra empresa cadastrada também vê continuam para ela. O histórico das manifestações enviadas (`nfe_manifestacoes`, com o `tpAmb` de cada envio) é mantido, e as manifestações registradas na SEFAZ não são afetadas. Um bloqueio da SEFAZ em vigor continua valendo, e os XMLs baixados ficam no armazenamento de blobs.

A distribuição exige o código IBGE da UF da empresa (`cUFAutor`). Cadastre a UF com `nanci company update --cnpj <CNPJ> --uf SP`; sem ela, `nfe pull` falha antes de pedir a senha.

## Requisitos de TLS

- **TLS 1.2.** Os servidores do AN não negociam TLS 1.3.
- **Certificado do cliente por renegociação.** O servidor (IIS) não pede o certificado no primeiro handshake; ele renegocia a conexão ao receber a requisição para um `.asmx` real e responde 403 se o cliente não apresentar certificado. O transporte comum (`internal/foundation/httpclient.NewTransport`) permite renegociação, desliga HTTP/2 (a renegociação do Go só funciona em HTTP/1.1) e entrega o certificado quando o servidor pede.
- **Cadeias públicas.** Os certificados dos servidores são emitidos por cadeias públicas (GlobalSign e Let's Encrypt/ISRG na verificação de 23/09/2026), já confiáveis no sistema operacional. O Nanci não embute raízes ICP-Brasil e não desliga a verificação.

`nanci nfe testar-conexao` faz só o handshake TLS com o host de distribuição e fecha a conexão. Nenhuma requisição é enviada, então nada é descontado do limite por hora. Como a SEFAZ só pede o certificado do cliente na primeira consulta real, um teste bem-sucedido confirma a rede e a cadeia do servidor, mas não que a SEFAZ aceitará o certificado.

## Limites de consulta

A SEFAZ bloqueia o CNPJ que consulta demais com a rejeição **656 (consumo indevido)**. O Nanci aplica as regras antes de enviar:

| Regra | Comportamento |
|---|---|
| 20 consultas por hora | Orçamento por empresa, origem e ambiente, em janela móvel de 1 hora (`sync_requests`). Cada requisição é registrada antes do envio, então tentativas que falham também contam. Esgotado o orçamento, o pull para com `rate_budget` e a origem fica bloqueada até a consulta mais antiga da janela completar 1 hora. |
| Fila em dia | Com `cStat` 137 (nenhum documento) ou `ultNSU` igual a `maxNSU`, o pull para com `caught_up` e a próxima consulta só é permitida 1 hora depois. |
| `cStat` 656 | O pull para com `consumo_indevido` e espera 1 hora. O `ultNSU` devolvido só é adotado se avançar o cursor. |
| Intervalo entre páginas | 2 segundos entre requisições do mesmo pull. |

O bloqueio fica em `company_sync_sources.blocked_until` e aparece como "Próxima consulta permitida após" no `nfe pull` e no `nfe status` (`NextAllowedAt` na camada `app`). Um pull dentro desse intervalo é recusado antes de pedir a senha. Zerar o estado local de sincronização não remove o bloqueio, porque a SEFAZ continua bloqueando.

**Regra dos 60 dias.** O Ambiente Nacional só gera NSU para a raiz de CNPJ que consultou o `distNSU` nos últimos 60 dias. Depois de uma pausa maior, a primeira consulta devolve 137 e os documentos do período parado não chegam mais pela distribuição. A regra entrou na NT 2014.002 v1.10, em produção desde 10/11/2021; ela é conhecida só por relatos de terceiros, que concordam entre si, e não foi conferida no texto da NT. Quando a última consulta respondida pela SEFAZ (`sync_state.last_success_at` da origem `nfe` no ambiente atual) passa de **45 dias**, o `nfe status` mostra um aviso e a página de NF-e mostra a etiqueta "Sem consulta há N dias", com a explicação na dica (`NSUEmRisco` e `IdleDays` na camada `app`). Uma empresa que nunca sincronizou NF-e não recebe o aviso. Um `nfe pull` reinicia a contagem. O limite está em `sync.DistIdleWarningDays`. Consultas feitas por outro programa com o mesmo CNPJ, como o ERP do contador, também contam para a SEFAZ, mas o Nanci não as vê, então o aviso pode aparecer sem que haja risco real.

A distribuição nunca é reenviada automaticamente pelo transporte, já que cada envio conta no limite. Um documento que falha ao ser decodificado ou interpretado três vezes seguidas no mesmo NSU tem o XML guardado, é marcado como não suportado e o cursor avança, para que um único documento não trave a fila.

## Manifestação do Destinatário

Só a empresa **destinatária** de uma NF-e **autorizada** pode manifestar. Os quatro eventos:

| Evento | `tpEvento` | Natureza |
|---|---|---|
| Ciência da Operação | 210210 | Não conclusiva. Libera o XML completo na distribuição. |
| Confirmação da Operação | 210200 | Conclusiva. |
| Desconhecimento da Operação | 210220 | Conclusiva. |
| Operação não Realizada | 210240 | Conclusiva; exige justificativa de 15 a 255 caracteres. |

Manifestações conclusivas são definitivas na SEFAZ e o Nanci não as desfaz.

- **Ciência em lote.** Pode ser enviada para várias notas de uma vez (`nfe ciencia`), em lotes de até 20 eventos, pedindo a senha do certificado uma vez só. Sem confirmação explícita nada é enviado. Depois de registrada, o XML completo chega num pull seguinte e o resumo é substituído.
- **Conclusivas nota a nota.** `nfe manifestar` envia um evento por vez. Tipo, justificativa, papel e situação são validados antes de pedir a senha.

Resultados por nota: `registrada` (`cStat` 135/136), `já registrada` (573 duplicidade), `rejeitada` (outro `cStat`, com o `xMotivo` da SEFAZ) e `não enviada` (o lote não teve resposta). Um lote sem resposta HTTP é reenviado uma vez; se ele já tinha chegado, a SEFAZ responde 573 e a nota conta como já registrada. Se continuar sem resposta, o envio dos lotes seguintes é interrompido e as notas não enviadas podem ser enviadas de novo. Na linha de comando, `nfe ciencia` e `nfe manifestar` mostram a tabela de resultados e terminam com erro (código de saída diferente de zero) quando alguma nota fica rejeitada ou não enviada, ou quando o envio é interrompido.

Uma ciência respondida com 655 conta como rejeitada, com a mensagem "NF-e já possui manifestação conclusiva". A SEFAZ não registrou a ciência, então o Nanci não grava evento nem muda a manifestação da nota.

### Prazos

Os prazos contam da autorização da NF-e (ou da emissão, quando a data de autorização não está disponível). São só avisos; o Nanci **não bloqueia** nenhuma manifestação por prazo.

- **Ciência:** a nota sem nenhuma manifestação passa a ser marcada como "ciência atrasada" 10 dias após a autorização.
- **Conclusiva:** Confirmação, Desconhecimento e Operação não Realizada podem ser registradas em até 90 dias da autorização, com alerta a partir de 30 dias antes. A SEFAZ rejeita um evento fora do prazo com `cStat` 596, e o `xMotivo` é mostrado.
- **Confirmação tácita:** passados os 90 dias sem nenhum evento conclusivo, a operação é considerada ocorrida, com os mesmos efeitos da Confirmação da Operação. A Ciência da Operação não interrompe esse prazo nem impede a presunção. O Nanci mostra essas notas como "confirmada tacitamente" em `nfe pendentes` e na tela de pendências.

Fontes: cláusula 15ª-C do Ajuste SINIEF 07/05, na redação dos Ajustes SINIEF 11/22 e 14/26 (este em vigor desde 1º de junho de 2026), e a NT 2020.001 v1.60 para a rejeição 596.

As constantes ficam em `internal/nfe/manifestacao.go`.

### Assinatura

Cada evento é assinado no perfil XMLDSig da NF-e: assinatura envelopada sobre `infEvento`, canonicalização C14N 1.0 inclusiva, digest SHA-1, assinatura RSA-SHA1 e o certificado da empresa em `KeyInfo`. O Nanci escreve o `infEvento` já na forma canônica, então usa só a biblioteca padrão do Go (`crypto/sha1` e `crypto/rsa`), sem biblioteca de C14N. O formato foi conferido contra um evento de ciência real aceito pela SEFAZ: o `infEvento` gerado é idêntico byte a byte e produz o mesmo `DigestValue` (`TestInfEventoDigest_RealEvent` em `internal/sefaz/xmldsig_test.go`). O teste opcional `go test -tags xmlsec ./internal/sefaz/` verifica a assinatura com o `xmlsec1`.

## Uso pela linha de comando

Todos os subcomandos de `nanci nfe` recebem a empresa por `--cnpj` (`-c`).

```bash
# 1. Cadastrar a UF da empresa (obrigatória para a distribuição)
nanci.exe company update --cnpj 12345678000199 --uf SP

# 2. Testar certificado e TLS sem gastar consultas
nanci.exe nfe testar-conexao --cnpj 12345678000199

# 3. Baixar resumos, notas completas e eventos
nanci.exe nfe pull --cnpj 12345678000199
nanci.exe nfe status --cnpj 12345678000199

# 4. Listar (filtros: --competencia/-m, --situacao, --completude, --papel/-p, --manifestacao, --emitente, --chave, --nao-vistos)
nanci.exe nfe list --cnpj 12345678000199 --completude resumo -p destinatario

# 5. Ciência da Operação: primeiro a simulação, que não envia nada;
#    depois, conferida a lista, o envio com --confirmar
nanci.exe nfe ciencia --cnpj 12345678000199 --todos-resumos
nanci.exe nfe ciencia --cnpj 12345678000199 --todos-resumos --confirmar

# 6. Manifestação conclusiva de uma nota (tipos: confirmacao, desconhecimento, nao_realizada)
#    Mesma ordem: simulação, depois --confirmar
nanci.exe nfe manifestar --cnpj 12345678000199 --chave <CHAVE> --tipo nao_realizada \
  --justificativa "Mercadoria recusada no recebimento"
nanci.exe nfe manifestar --cnpj 12345678000199 --chave <CHAVE> --tipo nao_realizada \
  --justificativa "Mercadoria recusada no recebimento" --confirmar

# 7. Notas sem manifestação conclusiva, por prazo
nanci.exe nfe pendentes --cnpj 12345678000199 --vencendo-em 30

# 8. Exportar XML
nanci.exe nfe export zip --cnpj 12345678000199 --competencia 2026-09 -p destinatario --out nfe.zip
nanci.exe nfe export zip --cnpj 12345678000199 --chave <CHAVE> --chave <OUTRA_CHAVE> --out notas.zip
nanci.exe nfe export xml --cnpj 12345678000199 --chave <CHAVE> --out nota.xml

# 9. Redefinir as NF-e da empresa (apaga as notas dos dois ambientes)
nanci.exe nfe reset --cnpj 12345678000199
nanci.exe nfe reset --cnpj 12345678000199 --confirmar
```

- `nfe ciencia` aceita `--chave` (repetível) ou `--todos-resumos`, nunca os dois. `--todos-resumos` seleciona os resumos autorizados em que a empresa é destinatária e que ainda não têm manifestação. A simulação lista as notas elegíveis, as ignoradas com o motivo e os prazos de cada uma.
- `nfe manifestar` também é simulação sem `--confirmar`. `nao-realizada` continua aceito como sinônimo de `nao_realizada`. Depois de uma manifestação conclusiva, nenhuma outra conclusiva é aceita para a mesma nota.
- `nfe reset` sem `--confirmar` mostra quantas notas, eventos e marcas de exportação seriam removidos e não altera nada.
- `nfe export zip` grava `<competencia>/<papel>/<chave>-procNFe.xml` e os eventos completos em `<competencia>/<papel>/eventos/`, no arquivo de `--out` (`-o`, padrão `nfe.zip`). Filtra por `--competencia` (`-m`), `--papel` (`-p`) e `--chave` (repetível). Resumos ficam de fora, a menos que se passe `--incluir-resumos`; `--incremental` exporta só o que ainda não foi exportado, mudou (por exemplo, um resumo que virou completa) ou cuja quantidade de eventos mudou desde a última exportação (por exemplo, depois de um cancelamento); o documento volta com todos os seus eventos completos.
- `nfe export xml` grava o `procNFe` da `--chave`, ou o `resNFe` de um resumo, em `--out` (`-o`); sem `--out`, o arquivo é `<chave>.xml` na pasta atual.

## Aplicativo desktop

O menu lateral ganha a entrada "NF-e", com as abas **Notas** e **Pendências** e os botões "Sincronizar NF-e" e "Redefinir NF-e"; este pede confirmação e faz o mesmo que `nfe reset --confirmar`. Sob o título, uma linha mostra a empresa, o CNPJ e a competência escolhidos no menu lateral, ou "Todas as competências", e continua visível quando a janela estreita esconde o menu. Na aba Notas é possível selecionar notas e usar "Registrar ciência", que abre um diálogo com as notas elegíveis, as ignoradas com o motivo e uma caixa de reconhecimento obrigatória antes do envio. As manifestações conclusivas são feitas nota a nota por um diálogo que pede o tipo, a justificativa quando exigida e uma revisão final. Enquanto a origem estiver bloqueada (656, fila em dia ou limite por hora), um aviso mostra o horário da próxima consulta permitida e o botão de sincronizar fica desabilitado. O pedido de senha mostra a finalidade (por exemplo, "Sincronização NF-e" ou "Assinatura: Ciência da Operação (12 notas)").

## Modelo de dados

Migrações `007` a `020` em `internal/store/migrations_v2/`:

- `007`: separa o estado de sincronização por origem (`source` em `sync_state` e `sync_runs`) e cria `company_sync_sources` (carga inicial e bloqueio por origem) e `sync_requests` (orçamento de consultas por hora).
- `008`: cria as tabelas de NF-e descritas abaixo.
- `009`: adiciona `companies.uf`, a UF da empresa, enviada como `cUFAutor`.
- `010`: adiciona `sync_state.failed_nsu` e `failed_nsu_attempts`, a contagem de falhas no mesmo NSU.
- `011`: adiciona o `tpAmb` de cada envio de manifestação. Envios anteriores a ela ficam com `tpAmb` vazio.
- `012`: indexa `company_nfe_documents` por nota, para a redefinição de NF-e.
- `013`: renomeia `nfe_manifestations` para `nfe_manifestacoes` e o índice `idx_company_nfe_documents_viewed` para `idx_company_nfe_documents_viewed_at`, e recria `sync_requests` com o mesmo `CHECK` de origem das outras tabelas de sincronização.
- `014`: remove `companies.initial_sync_completed_at`. A carga inicial da NFS-e, mostrada na lista de empresas e usada pela trava da política inicial no desktop, passa a vir só de `company_sync_sources`.
- `015`: cria as tabelas de CT-e, descritas em [CTE_SEFAZ.md](CTE_SEFAZ.md#modelo-de-dados).
- `016`: adiciona `tp_amb` (`1`, `2` ou vazio) a `nfe_documents` e `nfe_events`. As notas existentes recebem o ambiente da empresa que as vê, que até então não podia mudar depois da primeira sincronização de NF-e; os eventos recebem o da nota de mesma chave. Notas que nenhuma empresa vê e eventos sem nota ficam vazios.
- `017`: adiciona `viewed_at` a `company_cte_documents`, descrita em [CTE_SEFAZ.md](CTE_SEFAZ.md#modelo-de-dados).
- `018`: adiciona `exported_events` a `company_nfe_export_marks` e `company_cte_export_marks`, a quantidade de eventos do documento na última exportação. As marcas existentes contam os eventos gravados antes delas.
- `019`: recria `company_sync_sources` e `sync_requests` com a coluna `environment`, que entra na chave de `company_sync_sources` e no índice `idx_sync_requests_window`. As linhas existentes recebem o ambiente atual da empresa.
- `020`: tira o prefixo `NFS` da chave das NFS-e guardadas com o `Id` do `infNFSe`, liga os seus eventos e recalcula o status, descrita em [NFSE_ADN.md](NFSE_ADN.md#chave-de-acesso).

Tabelas de NF-e:

- `nfe_documents`: uma linha por chave de acesso, com os campos extraídos, a situação (`autorizada`, `denegada`, `cancelada`), a completude (`resumo` ou `completa`), o `tpAmb` (`tp_amb`) e o hash do XML bruto. Uma completa nunca é substituída por um resumo, e a situação só piora (cancelada > denegada > autorizada).
- `company_nfe_documents`: a relação empresa ↔ nota, com papel, motivo da visibilidade, estado da manifestação, NSUs em que foi vista e `viewed_at`, preenchida quando a empresa marca a nota como vista. Nota sem `viewed_at` é nova, e `nfe list --nao-vistos` mostra só as novas. A sincronização não mexe na coluna, e a redefinição de NF-e apaga a relação junto com ela.
- `nfe_events`: uma linha por (chave, `tpEvento`, `nSeqEvento`), com o `tpAmb` (`tp_amb`) do evento. Um `resEvento` é trocado pelo `procEventoNFe` quando este chega, e um evento enviado pelo Nanci se junta à cópia que volta pela distribuição.
- `nfe_manifestacoes`: registro de cada envio de manifestação (lote, `tpAmb`, resultado, `cStat`, `xMotivo`, protocolo), inclusive falhas, para auditoria.
- `company_nfe_export_marks`: o que já foi exportado, com qual hash e quantos eventos a nota tinha (`exported_events`), para a exportação incremental. A nota volta a ficar pendente quando o hash ou a quantidade de eventos muda, então um evento gravado no mesmo segundo da exportação, ou enquanto o ZIP era escrito, entra na próxima. A contagem inclui os resumos de evento, que o ZIP não leva: um `resEvento` novo faz a nota sair de novo sem arquivo novo, o que é inofensivo. Fica uma lacuna: um `resEvento` que depois chega como `procEvento` atualiza o mesmo registro (mesma chave, `tpEvento` e `nSeqEvento`), sem mudar a contagem, então o evento completo só sai numa exportação sem `--incremental`.

O estado da manifestação em `company_nfe_documents` é derivado dos eventos registrados de autoria da empresa: o evento conclusivo mais recente vence; sem conclusivo, uma ciência deixa a nota como `ciencia`. Os XMLs brutos ficam no mesmo armazenamento de blobs da NFS-e.

## CNPJ alfanumérico

Conferido em 07/10/2026 contra os textos oficiais. Vale para NF-e, CT-e e NFS-e; fica aqui porque a regra da chave nasce na NT da NF-e.

O que os textos dizem:

- **NT Conjunta 2025.001 v1.00** (ENCAT e RFB, 25/04/2025; homologação 06/04/2026, produção 06/07/2026), para NF-e, NFC-e, CT-e, CT-e OS, GTV-e, MDF-e, BP-e, NF3e e NFCom ([Portal da NF-e](https://www.nfe.fazenda.gov.br/portal/exibirArquivo.aspx?conteudo=5ZkvIZt10mQ%3D)):
  - O CNPJ continua com 14 posições: as 12 primeiras aceitam letras maiúsculas e as 2 últimas, os dígitos verificadores, são numéricas (`[A-Z0-9]{12}[0-9]{2}`). O DV é o módulo 11 de sempre, com cada caractere valendo o código ASCII menos 48 (`A` = 17). Exemplo da NT: `12.ABC.345/01DE-35`.
  - A chave de acesso continua com 44 posições e **não** é convertida em número: as letras do CNPJ vão na própria chave, que segue `[0-9]{6}[A-Z0-9]{12}[0-9]{26}` (letras só nas posições 7 a 18).
  - O DV da chave troca cada um dos caracteres pelo código ASCII menos 48 e aplica o módulo 11 de sempre: pesos de 2 a 9 da direita para a esquerda, e o resultado 10 ou 11 vira 0 (rotina do Anexo II). Uma chave só com números mantém o DV de antes.
  - A NT cita a possível exclusão das letras I, O, U, Q e F, mas diz que ela "precisa ser confirmada".
  - O código de barras do DANFE e do DACTE passa a ser CODE-128 híbrido (C e A), porque o CODE-128C só codifica números.
- **NT 2014.002 v1.40** (distribuição da NF-e, produção desde 08/07/2026) muda os campos de CNPJ de numérico para caractere ([cópia de terceiros](https://www.reformatributaria.com/wp-content/uploads/2026/07/NT2014.002_v1.40-WsNFeDistribuicaoDFe-CNPJ-alfa.pdf), ver `docs/pesquisa/2026-10-07-sefaz-dfe.md`).
- **NT SE/CGNFS-e 009** (v1.0 de 04/06/2026 e v1.01 de 01/10/2026, [portal da NFS-e](https://www.gov.br/nfse/pt-br/biblioteca/documentacao-tecnica/rtc/nota-tecnica-009-se-cgnfs-e-v-1-01.pdf)) só muda o tipo de todos os campos CNPJ de N para C. Não fala da chave nem do DV dela. Os esquemas de produção restrita de 27/07/2026 (`esquemas-nfse-rtc-v1-01-20260727.zip`) trazem `TSCNPJ` `[0-9A-Z]{14}`, `TSIdNFSe` `NFS[0-9]{9}[0-9A-Z]{14}[0-9]{27}` (letras só na inscrição federal, posições 10 a 23 da chave), `TSIdDPS` com letras só quando o tipo de inscrição é 2 (CNPJ) e `TSChaveNFSe` `[0-9]{6}([0-9A-Z]{14})[0-9]{30}`. Os esquemas listados em "Documentação atual" (v1.01 de 09/02/2026) ainda são só numéricos.

O que foi conferido no código:

| Lugar | Situação |
|---|---|
| `internal/foundation/cnpj` | Já seguia a regra (ASCII menos 48, DV numérico), com teste do exemplo da NT. Sem mudança. |
| `internal/dfe.ParseAccessKey` (NF-e e CT-e) | Já aceitava letras no CNPJ da chave e calcula o DV com ASCII menos 48; o DV das chaves de teste confere com a rotina do Anexo II. Um CNPJ com letras precisa ter DV válido, o que também barra letras nas posições 19 e 20, como no padrão da NT. Sem mudança. |
| `internal/nfse.ParseAccessKey` | Aceitava só 50 dígitos, então uma NFS-e de prestador com CNPJ alfanumérico falhava na leitura e, depois de três tentativas, era marcada como não suportada. Passa a aceitar letras maiúsculas na inscrição federal, como o `TSIdNFSe`, e converte minúsculas. |
| Consulta Direta (`useQuery.ts`, `QueryPage.vue`) | A regra `^\d{50}$` virou `isChaveNFSe` em `utils/formatters.ts`, com o mesmo padrão. As sugestões de chave não apagam mais as letras. |
| Log exportado (`internal/desktop/logsanitize.go`) | A chave de NFS-e com letras passa a ter a inscrição federal mascarada. A de NF-e e CT-e já era. |
| `internal/store/schema.sql` | Nenhuma `CHECK` de tamanho ou de dígitos em chave ou CNPJ; as colunas são `TEXT`. Sem mudança. |
| `utils/formatters.ts` | `formatCpfCnpj` e `formatChaveDFe` já aceitavam letras. |
| CLI | `--chave` da NF-e e do CT-e passa por `dfe.ParseAccessKey`, e `--cnpj` por `cnpj.Validate`. Sem mudança. |
| `internal/sefaz`, certificado | O CNPJ vai como texto no `distDFeInt` e nos eventos, e o do certificado é lido sem descartar letras. Sem mudança. |

Em aberto:

- **DV da chave de NFS-e com letras.** Nenhum texto da NFS-e o define. O Nanci nunca conferiu o DV da chave de NFS-e e continua sem conferir.
- **`TSChaveNFSe` contradiz `TSIdNFSe`.** O primeiro põe as letras nas posições 7 a 20; o segundo e a regra de formação, nas posições 10 a 23. O Nanci segue o `TSIdNFSe`. Rever quando os esquemas com CNPJ alfanumérico chegarem à "Documentação atual".
- **Letras vedadas.** A NT 2025.001 v1.00 não confirma a exclusão de I, O, U, Q e F; o Nanci aceita qualquer letra de A a Z.
- **CNPJ alfanumérico solto no log exportado.** Fora de uma chave e sem pontuação, continua sem máscara, para não mascarar identificadores hexadecimais (decisão do teste `raw alphanumeric untouched`).
- **Código de barras.** O Nanci ainda não gera DANFE nem DACTE; o CODE-128 híbrido entra quando a exportação em PDF vier.
- **Sem chave real nos testes.** Os testes usam o CNPJ do exemplo da NT; não há XML real de emitente com CNPJ alfanumérico.

## Fora do escopo

- NFC-e (modelo 65), NFCom, NF3e e CF-e SAT.
- Importação de XML avulso.
- Recuperar notas emitidas pela própria empresa.

## Próximos passos

- Importação de XML avulso.

## Atribuição

Partes de `internal/sefaz` (montagem e leitura do `distDFeInt`, envelope SOAP, estruturas de resposta e o formato do assinador) foram adaptadas de [gonfe](https://github.com/mschunke/gonfe), sob licença MIT. Os arquivos adaptados trazem um cabeçalho de atribuição, e a licença está em [`third_party/gonfe/LICENSE`](../third_party/gonfe/LICENSE).

## Referências

- [Portal Nacional da NF-e](https://www.nfe.fazenda.gov.br/portal/principal.aspx) (Notas Técnicas, em especial a NT 2014.002 da distribuição DF-e).
- [Regras de consumo indevido para DF-e (NS Tecnologia)](https://blog.nstecnologia.com.br/regras-de-consumo-indevido-para-dfe/).
- [Documentação de métodos do sped-nfe](https://github.com/nfephp-org/sped-nfe/tree/master/docs/metodos), usada só como referência de comportamento.
