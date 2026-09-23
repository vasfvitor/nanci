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

As tabelas de NF-e não guardam o ambiente de cada nota. Por isso, depois da primeira sincronização de NF-e, o ambiente da empresa fica travado: notas de homologação se misturariam às pendências de produção. Para trocar de ambiente, redefina antes as NF-e da empresa com `nanci nfe reset --cnpj <CNPJ>` (sem `--confirmar` só mostra o que seria removido) ou com o botão "Redefinir NF-e" do aplicativo. A redefinição remove as notas da empresa, seus eventos e marcas de exportação, e volta o cursor ao NSU 0. Notas que outra empresa cadastrada também vê continuam para ela. O histórico das manifestações enviadas (`nfe_manifestations`, agora com o `tpAmb` de cada envio) é mantido, e as manifestações registradas na SEFAZ não são afetadas. Um bloqueio da SEFAZ em vigor continua valendo, e os XMLs baixados ficam no armazenamento de blobs. A NFS-e não tem essa trava, porque o estado de sincronização dela já é separado por ambiente.

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
| 20 consultas por hora | Orçamento por empresa e origem, em janela móvel de 1 hora (`sync_requests`). Cada requisição é registrada antes do envio, então tentativas que falham também contam. Esgotado o orçamento, o pull para com `rate_budget` e a origem fica bloqueada até a consulta mais antiga da janela completar 1 hora. |
| Fila em dia | Com `cStat` 137 (nenhum documento) ou `ultNSU` igual a `maxNSU`, o pull para com `caught_up` e a próxima consulta só é permitida 1 hora depois. |
| `cStat` 656 | O pull para com `consumo_indevido` e espera 1 hora. O `ultNSU` devolvido só é adotado se avançar o cursor. |
| Intervalo entre páginas | 2 segundos entre requisições do mesmo pull. |

O bloqueio fica em `company_sync_sources.blocked_until` e aparece como "Próxima consulta permitida após" no `nfe pull` e no `nfe status` (`NextAllowedAt` na camada `app`). Um pull dentro desse intervalo é recusado antes de pedir a senha. Zerar o estado local de sincronização não remove o bloqueio, porque a SEFAZ continua bloqueando.

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

Resultados por nota: `registrada` (`cStat` 135/136), `já registrada` (573 duplicidade), `rejeitada` (outro `cStat`, com o `xMotivo` da SEFAZ) e `não enviada` (o lote não teve resposta). Um lote sem resposta HTTP é reenviado uma vez; se ele já tinha chegado, a SEFAZ responde 573 e a nota conta como já registrada. Se continuar sem resposta, o envio dos lotes seguintes é interrompido e as notas não enviadas podem ser enviadas de novo.

Uma ciência respondida com 655 conta como rejeitada, com a mensagem "NF-e já possui manifestação conclusiva". A SEFAZ não registrou a ciência, então o Nanci não grava evento nem muda a manifestação da nota.

### Prazos

Os prazos contam da autorização da NF-e (ou da emissão, quando a data de autorização não está disponível). São só avisos; o Nanci **não bloqueia** nenhuma manifestação por prazo.

- **Ciência:** a nota sem nenhuma manifestação passa a ser marcada como "ciência atrasada" 10 dias após a autorização.
- **Conclusiva:** Confirmação, Desconhecimento e Operação não Realizada podem ser registradas em até 90 dias da autorização, com alerta a partir de 30 dias antes. A SEFAZ rejeita um evento fora do prazo com `cStat` 596, e o `xMotivo` é mostrado.
- **Confirmação tácita:** passados os 90 dias sem nenhum evento conclusivo, a operação é considerada ocorrida, com os mesmos efeitos da Confirmação da Operação. A Ciência da Operação não interrompe esse prazo nem impede a presunção. O Nanci mostra essas notas como "confirmada tacitamente" em `nfe pendentes` e na tela de pendências.

Fontes: cláusula 15ª-C do Ajuste SINIEF 07/05, na redação dos Ajustes SINIEF 11/22 e 14/26 (este em vigor desde 1º de junho de 2026), e a NT 2020.001 v1.60 para a rejeição 596.

As constantes ficam em `internal/nfe/manifestation.go`.

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

# 4. Listar (filtros: --competencia, --situacao, --tipo, --papel, --manifestacao, --emitente, --chave, --nao-vistas)
nanci.exe nfe list --cnpj 12345678000199 --tipo resumo

# 5. Ciência da Operação: sem --confirmar só mostra o que seria enviado
nanci.exe nfe ciencia --cnpj 12345678000199 --todos-resumos
nanci.exe nfe ciencia --cnpj 12345678000199 --todos-resumos --confirmar

# 6. Manifestação conclusiva de uma nota (tipos: confirmacao, desconhecimento, nao-realizada)
nanci.exe nfe manifestar --cnpj 12345678000199 --chave <CHAVE> --tipo confirmacao
nanci.exe nfe manifestar --cnpj 12345678000199 --chave <CHAVE> --tipo nao-realizada \
  --justificativa "Mercadoria recusada no recebimento" --confirmar

# 7. Notas sem manifestação conclusiva, por prazo
nanci.exe nfe pendentes --cnpj 12345678000199 --vencendo-em 30

# 8. Exportar XML
nanci.exe nfe export zip --cnpj 12345678000199 --competencia 2026-09 --out nfe.zip
nanci.exe nfe export xml --cnpj 12345678000199 --chave <CHAVE>

# 9. Redefinir as NF-e da empresa (por exemplo, antes de trocar de ambiente)
nanci.exe nfe reset --cnpj 12345678000199
nanci.exe nfe reset --cnpj 12345678000199 --confirmar
```

- `nfe ciencia` aceita `--chave` (repetível) ou `--todos-resumos`, nunca os dois. `--todos-resumos` seleciona os resumos autorizados em que a empresa é destinatária e que ainda não têm manifestação. A simulação lista as notas elegíveis, as ignoradas com o motivo e os prazos de cada uma.
- `nfe manifestar` também é simulação sem `--confirmar`. Depois de uma manifestação conclusiva, nenhuma outra conclusiva é aceita para a mesma nota.
- `nfe reset` sem `--confirmar` mostra quantas notas, eventos e marcas de exportação seriam removidos e não altera nada.
- `nfe export zip` grava `<competencia>/<papel>/<chave>-procNFe.xml` e os eventos completos em `<competencia>/<papel>/eventos/`. Resumos ficam de fora, a menos que se passe `--incluir-resumos`; `--incremental` exporta só o que ainda não foi exportado ou mudou (por exemplo, um resumo que virou completa). `nfe export xml` grava o `procNFe`, ou o `resNFe` de um resumo.

## Aplicativo desktop

O menu lateral ganha a entrada "NF-e", com as abas **Notas** e **Pendências** e os botões "Sincronizar NF-e" e "Redefinir NF-e"; este pede confirmação e faz o mesmo que `nfe reset --confirmar`. Na aba Notas é possível selecionar notas e usar "Registrar ciência", que abre um diálogo com as notas elegíveis, as ignoradas com o motivo e uma caixa de reconhecimento obrigatória antes do envio. As manifestações conclusivas são feitas nota a nota por um diálogo que pede o tipo, a justificativa quando exigida e uma revisão final. Enquanto a origem estiver bloqueada (656, fila em dia ou limite por hora), um aviso mostra o horário da próxima consulta permitida e o botão de sincronizar fica desabilitado. O pedido de senha mostra a finalidade (por exemplo, "Sincronização NF-e" ou "Assinatura: Ciência da Operação (12 notas)").

## Modelo de dados

Migrações `007` a `011` em `internal/store/migrations_v2/`:

- `nfe_documents`: uma linha por chave de acesso, com os campos extraídos, a situação (`autorizada`, `denegada`, `cancelada`), a completude (`resumo` ou `completa`) e o hash do XML bruto. Uma completa nunca é substituída por um resumo, e a situação só piora (cancelada > denegada > autorizada).
- `company_nfe_documents`: a relação empresa ↔ nota, com papel, motivo da visibilidade, estado da manifestação, NSUs em que foi vista e marca de "vista".
- `nfe_events`: uma linha por (chave, `tpEvento`, `nSeqEvento`). Um `resEvento` é trocado pelo `procEventoNFe` quando este chega, e um evento enviado pelo Nanci se junta à cópia que volta pela distribuição.
- `nfe_manifestations`: registro de cada envio de manifestação (lote, `tpAmb`, resultado, `cStat`, `xMotivo`, protocolo), inclusive falhas, para auditoria. Envios anteriores à migração `011` ficam com `tpAmb` vazio.
- `company_nfe_export_marks`: o que já foi exportado e com qual hash, para a exportação incremental.
- `companies.uf`: a UF da empresa, enviada como `cUFAutor`.
- `sync_state.failed_nsu` e `failed_nsu_attempts`: contagem de falhas no mesmo NSU.

O estado da manifestação em `company_nfe_documents` é derivado dos eventos registrados de autoria da empresa: o evento conclusivo mais recente vence; sem conclusivo, uma ciência deixa a nota como `ciencia`. Os XMLs brutos ficam no mesmo armazenamento de blobs da NFS-e.

## Fora do escopo

- NFC-e (modelo 65), NFCom, NF3e e CF-e SAT.
- CT-e: próxima etapa.
- Importação de XML avulso.
- Recuperar notas emitidas pela própria empresa.

## Próximos passos

- CT-e pela mesma interface `Source` (`CTeDistribuicaoDFe`), com tabelas e tela próprias.
- Importação de XML avulso.
- Mascarar nos logs a chave de 50 caracteres da NFS-e, que também contém a inscrição do emitente (a chave de 44 caracteres da NF-e já é mascarada).
- Deixar de espelhar `companies.initial_sync_completed_at`: a coluna ainda é atualizada para a NFS-e porque a lista de empresas e a trava da política inicial no desktop a leem; a fonte da verdade é `company_sync_sources`.

## Atribuição

Partes de `internal/sefaz` (montagem e leitura do `distDFeInt`, envelope SOAP, estruturas de resposta e o formato do assinador) foram adaptadas de [gonfe](https://github.com/mschunke/gonfe), sob licença MIT. Os arquivos adaptados trazem um cabeçalho de atribuição, e a licença está em [`third_party/gonfe/LICENSE`](../third_party/gonfe/LICENSE).

## Referências

- [Portal Nacional da NF-e](https://www.nfe.fazenda.gov.br/portal/principal.aspx) (Notas Técnicas, em especial a NT 2014.002 da distribuição DF-e).
- [Regras de consumo indevido para DF-e (NS Tecnologia)](https://blog.nstecnologia.com.br/regras-de-consumo-indevido-para-dfe/).
- [Documentação de métodos do sped-nfe](https://github.com/nfephp-org/sped-nfe/tree/master/docs/metodos), usada só como referência de comportamento.
