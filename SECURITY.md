# Reportando uma Vulnerabilidade

O Nanci é uma aplicação **local-first**. Dados fiscais e certificados digitais são processados localmente e nunca são enviados a servidores de terceiros, exceto os webservices oficiais da Secretaria da Fazenda e Ambiente de Dados Nacional.

## Considerações de Segurança

- **SQLite sem criptografia em repouso**: O banco de dados SQLite do Nanci não possui criptografia em repouso. Os dados são armazenados em texto simples no arquivo de banco de dados local. Recomenda-se usar criptografia de disco do sistema operacional para proteger dados sensíveis.

- **Senhas de certificados**: As senhas dos certificados digitais **nunca são armazenadas** no banco de dados. Durante uma sessão de uso:
  - No **CLI**: A senha é solicitada via prompt interativo ou via variável de ambiente `NANCI_CERT_PASSWORD`, e descartada da memória após o uso.
  - No **Desktop**: A senha é solicitada via dialog frontend, passada por canal interno, e descartada da memória após o uso.
  - **Reutilização na sessão**: Após a primeira entrada bem-sucedida, a senha é armazenada automaticamente no keyring nativo do sistema operacional (Windows Credential Manager, macOS Keychain, Linux Secret Service). Nas operações subsequentes da **mesma sessão**, o app recupera a senha do keyring sem pedir novamente ao usuário.
  - **Ao fechar o app**: A senha fica salva no keyring do SO (não é apagada). Na próxima execução, o app tenta recuperá-la do keyring antes de pedir ao usuário.

Se você encontrar um problema de segurança relacionado à proteção local dos dados (SQLite), tratamento de certificados, injeção de dependências ou vazamento de logs locais, **por favor, não crie uma issue pública**.

Envie um e-mail com os detalhes do problema para `[sec@virtuaires.com.br]`.

### O que incluir:
- Um resumo do problema de segurança detectado.
- Os passos para reproduzir a falha.
- O impacto potencial, considerando o modelo de execução local do Nanci.
- Se possível, sugestões ou indicação de onde o código pode estar falhando.
