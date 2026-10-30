---
title: "Solução de problemas"
description: "Erros comuns, janela embaçada com dois monitores e como enviar logs."
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
| 593 ou 213 | O CNPJ da empresa não tem a mesma raiz do certificado ([NT 2014.002](https://www.nfe.fazenda.gov.br/portal/exibirArquivo.aspx?conteudo=uWO2d/gTuWg=), [NT 2020.001](https://www.nfe.fazenda.gov.br/portal/exibirArquivo.aspx?conteudo=%2BWTd7iuD21s%3D)). |
| 656: consumo indevido | Consultas demais ([NT 2014.002](https://www.nfe.fazenda.gov.br/portal/exibirArquivo.aspx?conteudo=uWO2d/gTuWg=)). Se outro sistema também consulta esse CNPJ na SEFAZ, eles somam. O Nanci espera uma hora. |
| 596 | A manifestação passou do prazo de 90 dias ([NT 2020.001](https://www.nfe.fazenda.gov.br/portal/exibirArquivo.aspx?conteudo=%2BWTd7iuD21s%3D)). |

## Janela levemente embaçada com dois monitores

Com o notebook ligado a um monitor externo, a janela pode ficar levemente embaçada, como se a resolução tivesse caído, depois de um diálogo abrir ou fechar ou quando o foco vai para o outro monitor.

1. Arraste a borda da janela para mudar o tamanho dela. Ou, em **Configurações → Sobre o Nanci**, clique em **Reajustar janela**, que faz o mesmo.
2. Se o texto voltar a ficar nítido, a falha é do WebView2, o componente do Windows que desenha a janela. A partir da versão 0.7.0, o WebView2 acompanha sozinho a escala de cada monitor; atualize o app.
3. Se continuar embaçado, ou se voltar a acontecer, abra uma issue com os dados de **Sobre o Nanci** (versão, WebView2, escala e monitores), a escala de cada monitor nas Configurações de vídeo do Windows, e o pacote de diagnóstico (veja abaixo). Diga também se o borrado some quando você clica de novo na janela do Nanci.

## Enviar logs

Em **Configurações**, clique em **Exportar Pacote de Diagnóstico (Logs)**. Os CNPJs saem mascarados. Na linha de comando, rode com `--verbose` e copie a saída.

Abra a issue em [github.com/vasfvitor/nanci/issues](https://github.com/vasfvitor/nanci/issues). **Nunca** anexe certificado, senha, XMLs reais ou o banco. Falhas de segurança vão por e-mail, conforme o [SECURITY.md](https://github.com/vasfvitor/nanci/blob/main/SECURITY.md).
