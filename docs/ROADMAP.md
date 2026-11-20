# Roteiro de longo prazo

Este arquivo ordena o trabalho que ainda não começou e registra o que foi descartado e por quê. O material bruto, com fontes e datas, está em `docs/pesquisa/`; confira a data antes de confiar num fato de lá. Itens pequenos sem plano ficam no `PENDENCIAS.md`. Ao concluir um item, apague-o daqui e registre as decisões no documento da área.

Esforço: **[P]** menos de uma hora, **[M]** alguns dias, **[G]** semanas.

## Issues abertas

- **#14** (links do site): rodar um verificador de links no HTML do Hugo e fechar ou corrigir. [P]
- **#4** (fork do go-pkcs12): o fork tem os backports até a v0.7.4 do upstream, e o upstream segue sem BER indefinido (conferido em 17/10/2026), então o fork fica. Ele também carrega `DecodeChainBytes` (#10). [M]
- **#15** (assinatura do instalador no Windows): depende de aprovação externa (SignPath); o passo no workflow é pequeno. [M]
- **#12** (criptografia em repouso): `modernc.org/sqlite` não tem SEE/SQLCipher, e os blobs são arquivos simples. Estacionado (07/10/2026) até o driver mudar ou surgir um pedido concreto.
- **#10** (senha na memória): a lista do que não dá para zerar está no `PENDENCIAS.md`.

## Mais dados fiscais por API

Universo levantado em 07/10/2026 (itens marcados *3P* vêm só de relatos de terceiros). Ordenado por valor para a empresa ou o contador × viabilidade para uma ferramenta desktop com certificado A1, em Go, gratuita.

| # | Fonte | O que traz | Acesso | Custo | Viabilidade | Valor |
|---|---|---|---|---|---|---|
| 1 | **MDF-e** `MDFeDistribuicaoDFe` v1.00 | MDF-e em que a empresa é autXML, contratante ou dona do veículo | A1, SOAP SVRS | grátis | médio | médio (transportadoras e contratantes) |
| 2 | NFS-e ADN `GET /NFSe/{chave}` | a nota pela chave | A1, já documentado | grátis | fácil | médio (consulta direta completa) |
| 3 | NFS-e São Paulo `ConsultaNFeRecebidas` | notas tomadas em SP, 50 por página | A1 + XMLDSig SHA-1 | grátis | médio | depende: *verificar primeiro* se SP já aparece no ADN pelo compartilhamento (LC 214 art. 62; *3P* diz que sim) |
| 4 | `CadConsultaCadastro4` | cadastro de IE do fornecedor por CNPJ | A1; BA, GO, MS, MT, PE, PR, RS, SP e SVRS; SP/ES podem recusar certificado de outra UF | grátis | fácil | baixo a médio (validar contraparte) |
| 5 | CNPJ público (`minhareceita.org`, BrasilAPI) | razão social, UF, situação | sem certificado | grátis, limites por IP | fácil | médio (preencher cadastro da empresa) |
| 6 | **eSocial** download de eventos | XML dos eventos já enviados, inclusive por contador | A1 da empresa; 10 chamadas/dia, 50 eventos por chamada, nada nos dias 1 a 7 | grátis | médio | alto para quem tem folha, mas é folha/RH: **fora do escopo** |
| 7 | Serpro Integra Contador | DCTFWeb, Simples/PGDAS, caixa postal, parcelamentos, pagamentos, DARF, CND | e-CNPJ mTLS + OAuth2 | pago por chamada (R$0,24 a 0,06), sem faixa grátis | médio | alto para contador. **Sem plano**; talvez um módulo opcional "traga suas credenciais" |
| 8 | CBS apuração (`api.receitafederal.gov.br/apuracao-cbs/v2`) | débitos e créditos de CBS | OAuth gov.br (não é A1) + **webhook HTTPS público** | grátis, 4/dia | difícil para desktop | alto a partir de 2027; acompanhar |
| 9 | EFD-Reinf consulta de recibos | só recibo, id e status | A1 | grátis | fácil | baixo |
| 10 | GNRE lote | emissão de guia | e-CNPJ + cadastro no portal | grátis | médio | baixo (emissão, não coleta) |
| 11 | Serpro Consulta NF-e / CNPJ / CND | JSON limitado pela anuência; CND em PDF | API key | R$0,66 a 0,88 por consulta | fácil | baixo (não devolve o XML) |

Sequência proposta: **2 e 5** são pequenas. **1 (MDF-e)** é a quarta origem e o gatilho que o `PENDENCIAS.md` prevê para reconsiderar um `DocumentTable` e um store comum. **3 e 4** quando houver um caso real (uma nota de SP que não apareça no ADN; um fornecedor a validar). **6 e 7** fora por decisão. **8** só com um caminho sem webhook público.

Sem caminho (não planejar):

- **NF-e e CT-e emitidos pela própria empresa**: a distribuição nunca devolve ao emitente (`consChNFe` responde 641). Só pela importação do XML.
- **NFC-e** para o comprador: não existe distribuição; desde 04/05/2026 compra com CNPJ exige NF-e 55.
- **NF3e, NFCom, BP-e, NFAg, NFGas**: sem serviço de distribuição na SVRS.
- **SPED (EFD, ECD, ECF)**: só pelo ReceitanetBX, sem API.
- **IBS (Comitê Gestor)**: nenhuma API encontrada.
- **CNDT e CRF/FGTS**: sem API oficial.
- **Open Finance**: exige autorização do Banco Central para receber dados.

Fatos que mudam prioridades (detalhes e fontes em `docs/pesquisa/2026-10-07-nfse-adn-municipios.md` e `2026-10-07-federal-serpro.md`):

- Desde 01/01/2026 todo município usa o emissor nacional ou compartilha com o ADN (LC 214/2025 art. 62), e ME e EPP usam o emissor nacional desde 01/11/2026 (CGSN 191/2026). APIs municipais quase não fazem falta.
- Não há biblioteca Go para DistDFe, CT-e, MDF-e, GNRE ou Integra Contador.

## Decisões

- APIs pagas (Serpro): sem plano; nada a preparar hoje (07/10/2026).
- eSocial: fora do escopo, por ser folha/RH e não documento fiscal (07/10/2026).
- Logs em disco mantêm o CNPJ em claro; só o pacote exportado mascara (07/10/2026, em `website/content/docs/privacidade.md`).
