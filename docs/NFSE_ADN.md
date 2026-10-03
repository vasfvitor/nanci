# Integração NFS-e e Ambiente de Dados Nacional (ADN)

Detalhamento técnico sobre como o Nanci consome a infraestrutura nacional do Ambiente de Dados Nacional (ADN).

---

## Endpoints Consumidos

O Nanci realiza chamadas HTTPS oficiais (com mTLS usando a chave privada do certificado correspondente à empresa) diretamente para os endpoints definidos pela Receita Federal e SERPRO:

### Consulta por NSU (Distribuição)

```text
GET DFe/{LastNSU}?cnpjConsulta={CNPJ}
```

O parâmetro `LastNSU` representa o cursor de sincronização. A resposta do governo contém um lote de notas fiscais geradas após esse número e os cursores atualizados (`ultNSU` e `maxNSU`).

### Consulta Direta

```text
GET NFSe/{ChaveAcesso}
GET NFSe/{ChaveAcesso}/Eventos
```

- `{ChaveAcesso}` deve possuir exatamente 50 dígitos numéricos correspondentes à chave de acesso da nota.

## Ambientes de Execução

1. **Produção (RFB):** Ambiente real contendo notas com validade fiscal jurídica. 
   - URL Base: `https://adn.nfse.gov.br/contribuintes`
2. **Produção Restrita (Homologação):** Ambiente para testes de desenvolvedores. Notas aqui não têm valor fiscal legal.
   - URL Base: `https://adn.producaorestrita.nfse.gov.br/contribuintes`

## Chave de acesso

A chave de acesso da NFS-e é guardada com os seus 50 dígitos. O parser lê o `chNFSe`; quando ele falta, usa o `Id` do `infNFSe`, que é `NFS` seguido dos 50 dígitos, sem o prefixo, e registra o aviso "document missing chNFSe; using infNFSe Id as fallback identifier". Um `Id` fora desse formato é guardado como veio.

Os eventos se ligam ao documento pelos 50 dígitos, e o status (`normal`, `cancelada`, `substituida`) é calculado a partir deles; uma substituição prevalece sobre um cancelamento. Ao marcar notas como vistas, uma chave com o prefixo `NFS` é aceita e usada como veio, para encontrar a linha que a migração `020` deixou com o prefixo (abaixo). Ao copiar a chave, o desktop remove o prefixo, porque o portal pede os 50 dígitos.

Versões anteriores guardavam o `Id` com o prefixo, e os eventos dessas notas nunca se ligavam a elas. A migração `020` reescreve essas chaves para os 50 dígitos, liga os eventos e recalcula o status, então um cancelamento antes ignorado passa a aparecer. Se outro documento já tem os mesmos 50 dígitos, a linha com prefixo fica como está e a tela mostra os seus 10 últimos caracteres. A migração não tem volta: os 50 dígitos são a chave canônica, e quais linhas tinham o prefixo não é guardado. Nas exportações, essas notas passam a sair como `<50 dígitos>.xml` e `<50 dígitos>.pdf` nos ZIPs, e com os 50 dígitos no CSV e na planilha.
