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
