---
title: "Integração NFS-e e ADN"
description: "Detalhes técnicos sobre os endpoints do Ambiente de Dados Nacional (ADN)."
weight: 60
icon: "server"
toc: true
---

## Endpoints de Consulta

As consultas diretas do desktop usam a base do ADN Contribuintes configurada para a empresa e chamam os endpoints oficiais por chave de acesso:

```text
GET /NFSe/{ChaveAcesso}
GET /NFSe/{ChaveAcesso}/Eventos
```

`{ChaveAcesso}` deve conter exatamente 50 dígitos.
