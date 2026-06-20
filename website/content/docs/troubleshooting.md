---
title: "Solução de Problemas"
description: "Erros comuns, logs e como resolver problemas no Nanci."
weight: 50
icon: "alert-triangle"
toc: true
---

## Erros da API do Ambiente de Dados Nacional (ADN)

### Erro E2220: "Nenhum documento localizado para o NSU informado"

Isso **não é necessariamente um erro de configuração do Nanci**. O erro E2220 indica que a conexão com o governo foi bem-sucedida, mas a partir do último NSU (Número Sequencial Único) salvo na sua máquina, não há nenhuma nota nova na fila de distribuição do seu CNPJ.

**O que fazer:** Apenas aguarde. O governo pode demorar para enfileirar notas novas. Se você acabou de emitir uma nota contra o seu CNPJ, ela pode não cair na fila de distribuição imediatamente. Leia mais sobre isso na [FAQ](faq).

### Erros de Autenticação / mTLS (Status 401 ou 403)

Isso geralmente ocorre se o seu CNPJ não tem permissão para consultar dados.

- O certificado está expirado?
- Você está tentando puxar dados de um CNPJ raiz diferente do certificado?

## Erros Locais do Aplicativo

### Falso Positivo de Antivírus (Windows Defender)

Como o Nanci foi escrito na linguagem Go (Golang) e não é um software com assinaturas digitais pagas extremamente conhecidas pelos fornecedores de antivírus tradicionais, ele pode ocasionalmente ser classificado como "software malicioso desconhecido" pelo Windows SmartScreen ou antivírus locais.
Isso ocorre com muitos executáveis novos da comunidade open source.

**O que fazer:** Se você baixou através da aba oficial de "Releases" do GitHub deste projeto (e conferiu o SHA-256 Checksum), você pode adicionar uma exclusão no seu antivírus ou clicar em "Mais informações -> Executar assim mesmo". O código do Nanci é inteiramente aberto e você pode compilá-lo você mesmo se preferir ter a máxima garantia. Ou mesmo pedir para um agente de IA validar o código fonte para você.

### Erro "MAC Verification failed" ao carregar Certificado

A senha do arquivo `.pfx` ou `.p12` digitada está incorreta. Se você alterou a senha recentemente ou se equivocou, vá na tela de Configurações ou recadastre o certificado com a senha correta.

## Onde encontro os Logs?

Se um problema persistir e não estiver listado aqui, você precisará dos logs para abrir uma issue (mas **lembre-se de ocultar dados sensíveis** do arquivo antes de nos mandar).

- No Desktop: Vá em Configurações e clique em **Exportar logs de diagnóstico (ZIP)**.
- Se estiver rodando via CLI, verifique o terminal ou os arquivos `nanci.log`/`nanci-desktop.log` na [pasta de dados do usuário](privacidade).
