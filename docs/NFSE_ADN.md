# Integração NFS-e e Ambiente de Dados Nacional (ADN)

Detalhamento técnico sobre como o Nanci consome a infraestrutura nacional do Ambiente de Dados Nacional (ADN).

---

## Endpoints Consumidos

O Nanci realiza chamadas HTTPS oficiais (com mTLS usando a chave privada do certificado correspondente à empresa) diretamente para os endpoints definidos pela Receita Federal e SERPRO. Fonte: [Manual dos Contribuintes, Guia para utilização das APIs do ADN](https://www.gov.br/nfse/pt-br/biblioteca/documentacao-tecnica/documentacao-atual/manual-contribuintes-apis-adn-sistema-nacional-nfse.pdf), versão 1.0 de 12/02/2026, conferido em 20/11/2026; o Swagger de produção restrita fica em `https://adn.producaorestrita.nfse.gov.br/contribuintes/docs/index.html`.

### Consulta por NSU (Distribuição)

```text
GET DFe/{LastNSU}?cnpjConsulta={CNPJ}
```

O parâmetro `LastNSU` representa o cursor de sincronização. A resposta do governo contém um lote de notas fiscais geradas após esse número e os cursores atualizados (`ultNSU` e `maxNSU`).

### Consulta Direta

```text
GET NFSe/{ChaveAcesso}/Eventos
```

- `{ChaveAcesso}` deve possuir exatamente 50 posições, a chave de acesso da nota: números, e letras maiúsculas só na inscrição federal quando o emitente tem CNPJ alfanumérico (ver [Chave de acesso](#chave-de-acesso)).
- O ADN também oferece `GET NFSe/{ChaveAcesso}`, que devolve a nota; o Nanci não o chama hoje.

## DANFSe

A NT SE/CGNFS-e 008/2026 desligou a API de geração de DANFSe do ADN (prazo final 03/08/2026). O Nanci não usa essa API: gera o DANFSe localmente a partir do XML, com a biblioteca `go-danfse-v2` (`internal/danfse`), então não foi afetado.

## Ambientes de Execução

Produção (`https://adn.nfse.gov.br/contribuintes`) e Produção Restrita, o ambiente de testes (`https://adn.producaorestrita.nfse.gov.br/contribuintes`).

## Chave de acesso

A chave de acesso da NFS-e é guardada com os seus 50 dígitos, sem o prefixo `NFS` do `Id` do `infNFSe`.

A chave é `cMun(7) ambGer(1) tpInsc(1) inscFed(14) nNFSe(13) AAMM(4) cNum(9) DV(1)`. Com o CNPJ alfanumérico (NT SE/CGNFS-e 009), a inscrição federal, nas posições 10 a 23, pode ter letras maiúsculas; o padrão `TSIdNFSe` dos esquemas de 27/07/2026 é `NFS[0-9]{9}[0-9A-Z]{14}[0-9]{27}`. `nfse.ParseAccessKey` e a Consulta Direta aceitam esse formato, convertem minúsculas para maiúsculas e não conferem o dígito verificador. O que ficou em aberto está em [NFE_SEFAZ.md](NFE_SEFAZ.md#cnpj-alfanumérico).

Os eventos se ligam ao documento pelos 50 dígitos, e o status (`normal`, `cancelada`, `substituida`) é calculado a partir deles; uma substituição prevalece sobre um cancelamento.

A migração `020` reescreveu para os 50 dígitos as chaves que versões anteriores guardavam com o prefixo, e não tem volta: quais linhas tinham o prefixo não é guardado. Se outro documento já tinha os mesmos 50 dígitos, a linha com prefixo ficou como estava, e a tela mostra os seus 10 últimos caracteres.
