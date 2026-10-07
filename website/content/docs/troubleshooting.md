---
title: "Solução de problemas"
description: "Erros comuns e como enviar logs."
weight: 80
---

## Erros comuns

| Mensagem | O que fazer |
|---|---|
| SmartScreen: "O Windows protegeu o computador" | Confira o hash do instalador ([Instalação](../instalacao/#instalar)) e clique em **Mais informações → Executar assim mesmo**. |
| Senha incorreta ou "MAC verification failed" | Sincronize de novo e digite a senha certa. |
| Certificado não encontrado | O arquivo mudou de lugar. Atualize o caminho em **Credenciais**. |
| E2220: nenhum documento localizado | Não é erro: não há notas novas na fila do ADN. |
| 401 ou 403 | O certificado está vencido ou é de outra empresa. |
| "empresa sem UF cadastrada" | Informe a UF em **Empresas → Editar**. |
| 593 ou 631 | O CNPJ da empresa não tem a mesma raiz do certificado. |
| 656: consumo indevido | Consultas demais. Se outro sistema também consulta esse CNPJ na SEFAZ, eles somam. O Nanci espera uma hora. |
| 596 | A manifestação passou do prazo de 90 dias. |

## Enviar logs

Em **Configurações**, clique em **Exportar Pacote de Diagnóstico (Logs)**. Os CNPJs saem mascarados. Na linha de comando, rode com `--verbose` e copie a saída.

Abra a issue em [github.com/vasfvitor/nanci/issues](https://github.com/vasfvitor/nanci/issues). **Nunca** anexe certificado, senha, XMLs reais ou o banco. Falhas de segurança vão por e-mail, conforme o [SECURITY.md](https://github.com/vasfvitor/nanci/blob/main/SECURITY.md).
