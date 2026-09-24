# Integração CT-e e Ambiente Nacional da SEFAZ

Detalhamento técnico sobre como o Nanci baixa CT-e (modelo 57, inclusive o CT-e Simplificado), CT-e OS (modelo 67) e GTV-e (modelo 64) pela distribuição DF-e do Ambiente Nacional.

---

## O que é a distribuição de CT-e

A SEFAZ mantém no Ambiente Nacional (AN) o webservice **CTeDistribuicaoDFe**, com o mesmo desenho do `NFeDistribuicaoDFe` descrito em [NFE_SEFAZ.md](NFE_SEFAZ.md): cada documento recebe um NSU próprio do CNPJ consultado, o Nanci envia o último NSU que já leu (`distNSU/ultNSU`) e recebe até 50 documentos seguintes, com os cursores atualizados (`ultNSU` e `maxNSU`). A fila guarda os documentos por cerca de 3 meses, e a política inicial da empresa (`--sync-start-policy`) **não** se aplica ao CT-e: o primeiro pull traz o que a SEFAZ ainda tiver.

Diferenças em relação à NF-e:

- **Não há resumo.** O XML completo, com o protocolo de autorização, chega direto.
- **Não há manifestação** para liberar documentos. O único evento que o tomador pode enviar é a prestação do serviço em desacordo, que o Nanci ainda não envia (veja abaixo).
- **Não há consulta por chave.** O serviço só aceita `distNSU` e `consNSU`.

Schemas distribuídos e lidos pelo Nanci:

| Schema | Documento |
|---|---|
| `procCTe` | CT-e, modelo 57. Em algumas versões também traz o CT-e Simplificado. |
| `procCTeOS` | CT-e OS (outros serviços), modelo 67. |
| `procGTVe` | GTV-e (guia de transporte de valores), modelo 64. |
| `procCTeSimp` | CT-e Simplificado, modelo 57. |
| `procEventoCTe` | Eventos (veja "Eventos distribuídos"). |

O tipo do documento (`cte`, `cte_os`, `gtve`, `cte_simplificado`) vem do elemento que envolve o `infCte` (`CTe`, `CTeOS`, `GTVe`, `CTeSimp`), então um CT-e Simplificado entregue no schema `procCTe` também é reconhecido. Os layouts 2.00, 3.00 e 4.00 podem aparecer na janela de 3 meses; o Nanci lê os campos por sufixo de caminho e ignora os que não conhece. Qualquer outro schema é guardado sem leitura e marcado como não suportado.

## O que a empresa recebe, por papel

O AN distribui o CT-e ao remetente, ao destinatário, ao expedidor, ao recebedor, ao tomador (inclusive o tomador terceiro do `toma4`) e aos CNPJ/CPF do grupo `autXML`. O Nanci classifica o papel da empresa em cada documento (coluna `PAPEL` do `cte list`), na ordem abaixo; o primeiro papel que a empresa ocupa é o principal:

| Papel | O que chega |
|---|---|
| `tomador` | CT-e em que a empresa toma o serviço de transporte, apontada pelo indicador `toma` (remetente, expedidor, recebedor ou destinatário) ou informada como terceiro (`toma4`). É quem escritura o frete. |
| `destinatario` | CT-e em que a empresa é o destinatário da carga. |
| `remetente` | CT-e em que a empresa é o remetente da carga. |
| `expedidor` | CT-e em que a empresa entrega a carga ao transportador no lugar do remetente. |
| `recebedor` | CT-e em que a empresa recebe a carga no lugar do destinatário. |
| `emitente` | A SEFAZ não distribui ao emitente o CT-e que ele mesmo emitiu, só os eventos de MDF-e sobre ele. Um documento só aparece com esse papel quando chega por outro motivo (por exemplo, o CNPJ também está no `autXML`). |
| `autorizado` | CT-e em que o CNPJ ou CPF foi informado no grupo `autXML`. |
| `none` | A empresa só compartilha a raiz do CNPJ com alguma das partes, ou o motivo não foi identificado. |

Uma empresa pode ocupar vários papéis no mesmo CT-e, por exemplo remetente e tomador. Todos ficam em `papeis`, em ordem, e o filtro `--papel` casa com o papel principal ou com qualquer um deles: `--papel remetente` encontra o CT-e cujo papel principal é `tomador` quando a empresa também é o remetente.

O tomador é resolvido na leitura do XML:

- **CT-e:** `ide/toma3/toma` (`ide/toma03/toma` no layout 2.00) aponta para uma das partes: 0 remetente, 1 expedidor, 2 recebedor, 3 destinatário. O tomador recebe o CNPJ, o nome, a IE e a UF dessa parte. Com `toma4`, o tomador é a parte informada ali.
- **GTV-e:** `ide/toma/toma` aponta para o remetente (0) ou o destinatário (1); `tomaTerceiro` informa um terceiro.
- **CT-e OS e CT-e Simplificado:** o tomador vem sempre de `infCte/toma`.

O código bruto do indicador fica em `tomador_indicador` (vazio no CT-e OS). Um indicador que aponta para uma parte ausente do XML (o que pode acontecer num CT-e de anulação ou de complemento) deixa o tomador vazio, com um aviso de leitura, e a empresa cai no papel seguinte que ocupar.

Para terceiros do `autXML`, o AN troca as chaves de documentos relacionados (`infDoc`, `docAnt`, `refCTe`) por `9999…`. O Nanci descarta essas chaves (veja "Ligação com a NF-e").

Eventos de uma chave que a empresa ainda não tem localmente são descartados, exceto os que a própria empresa assinou, que ficam guardados e são ligados ao documento quando ele chegar.

## Endpoints Consumidos

A chamada é SOAP 1.2 síncrona sobre HTTPS com o certificado A1 da empresa, método `cteDistDFeInteresse`, pedido `distDFeInt` versão `1.00` no namespace `http://www.portalfiscal.inf.br/cte`.

1. **Produção:** `https://www1.cte.fazenda.gov.br/CTeDistribuicaoDFe/CTeDistribuicaoDFe.asmx`
2. **Homologação:** `https://hom1.cte.fazenda.gov.br/CTeDistribuicaoDFe/CTeDistribuicaoDFe.asmx`

O ambiente segue o da empresa, como na NF-e: `producao` usa produção (`tpAmb` 1) e `producao_restrita` usa homologação (`tpAmb` 2). As URLs estão em `internal/sefaz/endpoints.go` e só podem ser substituídas pelo código (`sefaz.ClientConfig.Endpoints`). A action SOAP (`http://www.portalfiscal.inf.br/cte/wsdl/CTeDistribuicaoDFe/cteDistDFeInteresse`) segue a do serviço de NF-e e foi aceita pelo serviço de produção em 24/09/2026 (resposta `cStat` 137 para uma empresa sem CT-e na fila).

Cada CT-e e cada evento guardam o seu `tpAmb` (`tp_amb`): o do XML (`ide/tpAmb` no documento, `infEvento/tpAmb` no evento). Um XML de um ambiente diferente do consultado é guardado com o `tpAmb` dele e recebe um aviso de leitura. A lista, o `cte status` e a exportação mostram só os documentos do ambiente atual da empresa. O CT-e não trava a troca de ambiente: a empresa pode mudar de ambiente a qualquer momento, os documentos do outro ambiente deixam de aparecer e voltam quando ela retorna a ele. O cursor de sincronização é separado por ambiente. O bloqueio da SEFAZ (`company_sync_sources.blocked_until`) e o orçamento por hora, não: valem para a empresa e a origem `cte` em qualquer ambiente, então um bloqueio recebido em produção também adia o próximo pull de CT-e em homologação.

Para apagar os CT-e da empresa, use `nanci cte reset --cnpj <CNPJ>` (sem `--confirmar` só mostra o que seria removido) ou o botão "Redefinir CT-e" do aplicativo. A redefinição remove os documentos da empresa nos dois ambientes, seus eventos e marcas de exportação, e volta o cursor ao NSU 0. Documentos que outra empresa cadastrada também vê continuam para ela, assim como os eventos assinados por outra empresa cadastrada. Um bloqueio da SEFAZ em vigor continua valendo, e os XMLs baixados ficam no armazenamento de blobs.

A distribuição exige o código IBGE da UF da empresa (`cUFAutor`). Cadastre a UF com `nanci company update --cnpj <CNPJ> --uf SP`; sem ela, `cte pull` falha antes de pedir a senha.

## Requisitos de TLS

Os mesmos da NF-e (veja [NFE_SEFAZ.md](NFE_SEFAZ.md#requisitos-de-tls)): TLS 1.2 (os servidores não negociam TLS 1.3), certificado do cliente pedido por renegociação e cadeias públicas. Na verificação de 23/09/2026, `www1.cte.fazenda.gov.br` e `hom1.cte.fazenda.gov.br` apresentavam certificados da AC SERPRO sob a raiz GlobalSign Root R46, já confiável no sistema operacional. O transporte e a verificação são os mesmos da NF-e.

`nanci cte testar-conexao` faz só o handshake TLS com o host de distribuição de CT-e e fecha a conexão. Nenhuma requisição é enviada, então nada é descontado do limite por hora. Como na NF-e, um teste bem-sucedido confirma a rede e a cadeia do servidor, mas não que a SEFAZ aceitará o certificado.

## Limites de consulta

A SEFAZ bloqueia o CNPJ que consulta demais com a rejeição **656 (consumo indevido)**. O Nanci aplica ao CT-e as mesmas regras da NF-e, com orçamento e bloqueio próprios da origem `cte`:

| Regra | Comportamento |
|---|---|
| 20 consultas por hora | Orçamento por empresa e origem, em janela móvel de 1 hora (`sync_requests`), contado separado do da NF-e. Cada requisição é registrada antes do envio, então tentativas que falham também contam. Esgotado o orçamento, o pull para com `rate_budget` e a origem fica bloqueada até a consulta mais antiga da janela completar 1 hora. |
| Fila em dia | Com `cStat` 137 (nenhum documento) ou `ultNSU` igual a `maxNSU`, o pull para com `caught_up` e a próxima consulta só é permitida 1 hora depois. |
| `cStat` 656 | O pull para com `consumo_indevido` e espera 1 hora. O `ultNSU` devolvido só é adotado se avançar o cursor. |
| Intervalo entre páginas | 2 segundos entre requisições do mesmo pull. |

A NT 2015.002 não publica um limite de consultas por hora para o CT-e; o Nanci usa o da NF-e por prudência. Se a SEFAZ contar as consultas de NF-e e de CT-e juntas por CNPJ, um pull de CT-e logo depois de um de NF-e pode receber 656. O bloqueio é gravado e respeitado, então o efeito é uma hora de espera.

O bloqueio aparece como "Próxima consulta permitida após" no `cte pull` e no `cte status`. Um pull dentro desse intervalo é recusado antes de pedir a senha. A distribuição nunca é reenviada automaticamente, e um documento que falha ao ser decodificado ou interpretado três vezes seguidas no mesmo NSU tem o XML guardado, é marcado como não suportado e o cursor avança.

## Eventos distribuídos

| Evento | `tpEvento` | Tipo no Nanci |
|---|---|---|
| Cancelamento | 110111 | `cancelamento` |
| Carta de correção | 110110 | `carta_correcao` |
| EPEC | 110113 | `epec` |
| Registro multimodal | 110160 | `registro_multimodal` |
| GTV | 110170 | `gtv` |
| Comprovante de entrega | 110180 | `comprovante_entrega` |
| Cancelamento do comprovante de entrega | 110181 | `cancelamento_comprovante_entrega` |
| Insucesso na entrega | 110190 | `insucesso_entrega` |
| Cancelamento do insucesso na entrega | 110191 | `cancelamento_insucesso_entrega` |
| Prestação do serviço em desacordo | 610110 | `prestacao_desacordo` |
| Cancelamento da prestação em desacordo | 610111 | `cancelamento_desacordo` |
| MDF-e autorizado | 310610 | `mdfe_autorizado` |
| MDF-e cancelado | 310611 | `mdfe_cancelado` |

Outro `tpEvento` é guardado como `unknown`. Um evento só conta como registrado quando o `retEventoCTe` traz `cStat` 134, 135 ou 136; os outros são guardados com um aviso de leitura e não mudam a situação do documento. Um cancelamento registrado deixa o CT-e como `cancelada`. Os campos de texto do evento (`descEvento`, `xJust`, `xObs`, `xCondUso`) são guardados, e a carta de correção vira um texto `grupo.campo=valor; …`.

Eventos de uma chave que a empresa não tem localmente são descartados com um registro no log, salvo quando o autor é a própria empresa. Na prática isso descarta os eventos de MDF-e que chegam ao emitente sobre o próprio CT-e, que a distribuição não entrega a ele.

## Prestação do serviço em desacordo

O evento 610110 é a forma de o **tomador** declarar que o serviço de transporte não foi prestado como consta no CT-e. Só o tomador pode enviá-lo, em até 45 dias da autorização, com uma observação de 15 a 255 caracteres, e o envio vai à **SEFAZ autorizadora do CT-e** (serviço de recepção de eventos da UF), não ao Ambiente Nacional. O Nanci mostra o evento quando ele chega pela distribuição, mas não o envia. Veja "Próximos passos".

## Ligação com a NF-e

As chaves das NF-e transportadas ficam em `nfe_chaves`: `infCTeNorm/infDoc/infNFe/chave` nos layouts 3.00 e 4.00, `rem/infNFe/chave` no 2.00 e `det/infNFe/chNFe` no CT-e Simplificado. Só entram chaves que passam na validação da chave de acesso (44 dígitos e dígito verificador), sem repetição. As chaves `9999…` que o AN envia aos terceiros do `autXML` são descartadas, com um único aviso de leitura com a contagem; para esses documentos a ligação com a NF-e não existe.

`nanci cte list --nfe <CHAVE_NFE>` (e o filtro de chave de NF-e no aplicativo) encontra os CT-e que transportaram uma NF-e. As telas ainda não mostram a ligação entre um CT-e e as notas dele.

## Uso pela linha de comando

Todos os subcomandos de `nanci cte` recebem a empresa por `--cnpj` (`-c`).

```bash
# 1. Cadastrar a UF da empresa (obrigatória para a distribuição)
nanci.exe company update --cnpj 12345678000199 --uf SP

# 2. Testar certificado e TLS sem gastar consultas
nanci.exe cte testar-conexao --cnpj 12345678000199

# 3. Baixar documentos e eventos
nanci.exe cte pull --cnpj 12345678000199
nanci.exe cte status --cnpj 12345678000199

# 4. Listar (filtros: --competencia/-m, --situacao, --papel/-p, --modelo, --emitente, --tomador, --nfe, --chave)
nanci.exe cte list --cnpj 12345678000199 -m 2026-09 -p tomador
nanci.exe cte list --cnpj 12345678000199 --modelo 67

# 5. CT-e que transportaram uma NF-e
nanci.exe cte list --cnpj 12345678000199 --nfe <CHAVE_NFE>

# 6. Exportar XML
nanci.exe cte export zip --cnpj 12345678000199 --competencia 2026-09 -p tomador --out cte.zip
nanci.exe cte export zip --cnpj 12345678000199 --incremental
nanci.exe cte export xml --cnpj 12345678000199 --chave <CHAVE> --out cte.xml

# 7. Redefinir os CT-e da empresa (apaga os documentos dos dois ambientes)
nanci.exe cte reset --cnpj 12345678000199
nanci.exe cte reset --cnpj 12345678000199 --confirmar
```

- `cte status` mostra cursor, bloqueio, consultas da última hora e os totais do ambiente atual por papel principal: tomador, destinatário, remetente e os demais.
- `cte list` mostra chave, modelo, número e série, emissão, emitente, tomador, papel, valor da prestação e situação. `--papel` casa com o papel principal ou com qualquer outro papel da empresa no documento; `--modelo` aceita 57, 64 ou 67; `--chave` pode repetir.
- `cte export zip` grava `<competencia>/<papel>/<chave>-procCTe.xml` (`-procCTeOS`, `-procGTVe` ou `-procCTeSimp` conforme o tipo) e os eventos em `<competencia>/<papel>/eventos/`, no arquivo de `--out` (padrão `cte.zip`). Filtra por `--competencia` (`-m`), `--papel` (`-p`) e `--chave` (repetível); `--incremental` exporta só o que ainda não foi exportado ou mudou.
- `cte export xml` grava o XML do documento da `--chave` em `--out` (`-o`), por padrão `<chave>.xml`.
- `cte reset` sem `--confirmar` mostra quantos documentos, eventos e marcas de exportação seriam removidos e não altera nada.

## Aplicativo desktop

O menu lateral ganha a entrada "CT-e", logo abaixo de "NF-e". A página tem uma única tabela, sem aba de pendências, com filtros de competência, papel, modelo, situação, tomador e chave de NF-e, e os botões "Sincronizar CT-e", "Exportar" e "Redefinir CT-e"; este pede confirmação e faz o mesmo que `cte reset --confirmar`. Os eventos de cada documento abrem num diálogo. Enquanto a origem estiver bloqueada (656, fila em dia ou limite por hora), um aviso mostra o horário da próxima consulta permitida e o botão de sincronizar fica desabilitado. O pedido de senha mostra a finalidade "Sincronização CT-e", o que separa os pedidos quando NFS-e, NF-e e CT-e sincronizam ao mesmo tempo.

## Modelo de dados

A migração `015` em `internal/store/migrations_v2/` cria as tabelas de CT-e. A origem `cte` já era aceita pelos `CHECK` de `sync_state`, `sync_runs`, `company_sync_sources` (migração `007`) e `sync_requests` (migração `013`).

- `cte_documents`: uma linha por chave de acesso, com `tp_amb`, `modelo` (`57`, `64`, `67`), `tipo_documento` (`cte`, `cte_os`, `gtve`, `cte_simplificado`), série, número, CFOP, natureza da operação, emissão, competência, autorização e protocolo; `tp_cte`, `tp_serv` e `modal` como texto bruto, sem `CHECK`, porque os domínios mudaram entre os layouts 3.00 e 4.00; municípios e UF de início e fim da prestação; CNPJ/CPF e nome do emitente, remetente, destinatário, expedidor, recebedor e tomador (IE e UF só do emitente e do tomador); `tomador_indicador`; `autorizados_cnpj` e `nfe_chaves` (listas separadas por vírgula); valores em centavos (`total_value` do `vTPrest`, `receivable_value` do `vRec`, `icms_value`, `tot_trib_value`, `carga_value`); produto predominante; situação; versão do layout; hash do XML bruto e avisos de leitura.
- `company_cte_documents`: a relação empresa ↔ documento, com o papel principal (`company_role`), todos os papéis (`papeis`, separados por vírgula na ordem de classificação), o motivo da visibilidade (`exact_<papel>`, `same_root_only`, `unknown`) e os NSUs em que foi visto.
- `cte_events`: uma linha por (chave, `tpEvento`, `nSeqEvento`), com `tp_amb`, órgão, tipo, datas do evento e do registro, `registered`, `cStat`, `xMotivo`, protocolo, autor e os textos do evento. Um evento pode chegar antes do documento e é ligado a ele quando o documento chega.
- `company_cte_export_marks`: o que já foi exportado e com qual hash, para a exportação incremental.

A situação é `autorizada`, `denegada` ou `cancelada`. Vem do `cStat` do protocolo (100 e 150 → `autorizada`; 110, 205, 301, 302 e 303 → `denegada`) e passa a `cancelada` quando há um cancelamento registrado. Ela só piora (cancelada > denegada > autorizada) e é recalculada a partir de `cte_events` a cada evento gravado. Um protocolo com outro `cStat` é tratado como falha de leitura. Os códigos de denegação seguem os da NF-e e ainda precisam ser conferidos no MOC CT-e 4.00.

Um CT-e recebido de novo substitui o anterior (não há resumo a preservar), mantendo o identificador e a situação mais grave. Os XMLs brutos ficam no mesmo armazenamento de blobs da NFS-e e da NF-e. No ZIP de exportação, cada documento vai para `<competencia>/<papel>/<chave>-procCTe.xml` e seus eventos para `<competencia>/<papel>/eventos/<chave>-<tpEvento>-<nSeqEvento>.xml`; o papel é o principal, e `none` vira a pasta `sem-papel-fiscal`, como na NF-e.

## Fora do escopo

- Envio da prestação do serviço em desacordo (610110) e do seu cancelamento.
- MDF-e: os documentos em si e os eventos de MDF-e sobre CT-e que a empresa não tem.
- Recuperar CT-e emitidos pela própria empresa.
- Ligação visual entre CT-e, NF-e e NFS-e nas telas.

## Próximos passos

- Prestação do serviço em desacordo: tabela UF → SEFAZ autorizadora do CT-e por ambiente (SVRS para a maioria das UF; SP, MT, MS, MG e PR com endpoint próprio), com a UF tirada da chave; generalizar o assinador de `internal/sefaz/xmldsig.go` para o `eventoCTe` 4.00; tabela de auditoria dos envios, como `nfe_manifestacoes`; confirmação explícita no CLI e no aplicativo, com o prazo exibido.
- Mostrar a ligação CT-e ↔ NF-e nas telas das duas origens, a partir de `nfe_chaves`.

## Atribuição

O CT-e usa a mesma montagem do `distDFeInt`, o mesmo envelope SOAP e a mesma leitura do `retDistDFeInt` da NF-e, adaptados de [gonfe](https://github.com/mschunke/gonfe). Veja a seção "Atribuição" de [NFE_SEFAZ.md](NFE_SEFAZ.md#atribuição).

## Referências

- [Portal Nacional do CT-e](https://www.cte.fazenda.gov.br/portal/) (NT 2015.002 v1.05, distribuição DF-e de CT-e, e MOC CT-e 4.00).
- [sped-cte](https://github.com/nfephp-org/sped-cte), usado só como referência de comportamento.
- [NFE_SEFAZ.md](NFE_SEFAZ.md), para as regras compartilhadas com a NF-e.
