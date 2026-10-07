---
title: "Solução de problemas"
description: "Erros comuns do ADN, da SEFAZ e do certificado, e como coletar logs para relatar um problema."
weight: 80
---

## Instalação

### Aviso do SmartScreen ou do antivírus

O Nanci não tem assinatura de código paga, então o Windows SmartScreen pode mostrar "O Windows protegeu o computador" e alguns antivírus podem desconfiar de um executável novo.

1. Baixe só da página de [releases do GitHub](https://github.com/vasfvitor/nanci/releases).
2. Confira o SHA-256 do instalador com o arquivo `nanci-checksums.txt` da mesma versão (veja [Instalação](../instalacao/#instalar)). Cada release também tem atestado de proveniência do GitHub, que liga o instalador ao commit que o gerou.
3. Se os valores baterem, clique em **Mais informações → Executar assim mesmo**.

Se preferir, compile a partir do código-fonte (veja [Desenvolvimento](../desenvolvimento/)).

## Certificado

### Senha incorreta ou "MAC verification failed"

A senha não abre o arquivo `.pfx`/`.p12`. Sincronize de novo e informe a senha correta quando ela for pedida. Se uma senha antiga estiver guardada, o Nanci percebe que ela não abre mais o certificado e pede outra.

### Arquivo do certificado não encontrado

O Nanci guarda o caminho do arquivo, não uma cópia. Se o certificado foi movido ou renovado, atualize o caminho na tela **Credenciais**.

### Certificado vencido

O ADN e a SEFAZ recusam certificados vencidos. Cadastre o certificado novo em **Credenciais** e ligue-o à empresa.

## NFS-e (ADN)

### E2220: "Nenhum documento localizado para o NSU informado"

Não é erro. A conexão funcionou, mas não há documentos novos depois do último NSU lido. Se você esperava notas, veja a [FAQ](../faq/#emito-nfs-e-todo-mês-mas-o-nanci-não-baixou-nada).

### Status 401 ou 403

O ADN recusou o certificado para esse CNPJ. Confira se o certificado está válido e se é da mesma empresa (ou da matriz) que está sendo consultada.

## NF-e e CT-e (SEFAZ)

### "empresa sem UF cadastrada"

A distribuição da SEFAZ exige a UF. Informe-a em **Empresas → Editar** ou com `nanci company update --cnpj <CNPJ> --uf SP`.

### 593 ou 631: CNPJ diferente do certificado

A raiz do CNPJ (8 primeiros dígitos) da empresa não é a mesma do certificado. Use o certificado da própria empresa ou da matriz.

### 656: consumo indevido

A SEFAZ bloqueou o CNPJ por consultar demais. Isso pode acontecer quando outro sistema (o ERP, por exemplo) também consulta a distribuição para o mesmo CNPJ, já que o limite é da SEFAZ e não do Nanci. O Nanci espera 1 hora antes da próxima consulta. Evite que dois sistemas consultem o mesmo CNPJ.

### 403 na primeira consulta, mas Testar Conexão funcionou

O teste confirma a rede e a cadeia do servidor, mas a SEFAZ só pede o certificado do cliente na consulta real. Um 403 nessa hora indica que ela recusou o certificado: confira validade, raiz do CNPJ e se é um e-CNPJ A1.

### 596: evento fora do prazo

A manifestação conclusiva foi enviada depois de 90 dias da autorização da NF-e. Não há como registrá-la pela SEFAZ.

## Coletar logs para relatar um problema

- **Desktop**: em **Configurações**, clique em **Exportar Pacote de Diagnóstico (Logs)**. O ZIP leva os arquivos `nanci-desktop.log` com os CNPJs mascarados. **Abrir Pasta de Logs** mostra os arquivos originais. O **Console**, aberto pela barra de título, mostra os logs em tempo real.
- **Linha de comando**: rode o comando com `--verbose` (ou `--trace`, mais detalhado) e copie a saída do terminal.

{{< theme-image light="/img/screenshots/configuracoes-light.png" dark="/img/screenshots/configuracoes-dark.png" alt="Tela de configurações" >}}

Antes de anexar, confira o conteúdo. **Nunca** anexe o certificado, a senha, XMLs reais ou o banco `nanci-v1.db` numa issue pública. Abra a issue em [github.com/vasfvitor/nanci/issues](https://github.com/vasfvitor/nanci/issues). Problemas de segurança vão por e-mail, conforme a [política de segurança](https://github.com/vasfvitor/nanci/blob/main/SECURITY.md).
