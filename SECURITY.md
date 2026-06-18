# Reportando uma Vulnerabilidade

O Nanci é uma aplicação **local-first**. Dados fiscais e certificados digitais são processados localmente e nunca são enviados a servidores de terceiros, exceto os webservices oficiais da Secretaria da Fazenda e Ambiente de Dados Nacional.

## Considerações de Segurança

O Nanci usa um banco SQLite local. Por padrão, esse banco **não é criptografado em repouso**, então os dados ficam salvos em texto simples no arquivo da máquina.

As senhas dos certificados digitais **não são armazenadas no banco de dados do Nanci**. Elas podem ser solicitadas no uso via CLI, Desktop ou variável de ambiente `NANCI_CERT_PASSWORD`.

Quando uma senha é informada com sucesso, o Nanci pode salvá-la no **keyring nativo do sistema operacional** para evitar que o usuário precise digitá-la novamente nas próximas operações.

Durante o uso, a senha também pode existir temporariamente em memória. Há [planos para melhorar esse aspecto](https://github.com/vasfvitor/nanci/issues/10), mas não é garantido a remoção de todas as cópias temporárias criadas por bibliotecas, pelo sistema operacional ou pelo próprio runtime.

Até que seja implementado [criptografia em repouso](https://github.com/vasfvitor/nanci/issues/12), recomendamos usar criptografia de disco, usuário individual no sistema operacional e controle adequado de acesso à máquina. Em máquinas compartilhadas, o Nanci só deve ser usado quando todos os usuários com acesso ao perfil, disco ou backups também puderem acessar os dados locais do aplicativo.

Se você encontrar um problema de segurança relacionado à proteção local dos dados (SQLite), tratamento de certificados, injeção de dependências ou vazamento de logs locais, **por favor, não crie uma issue pública**.

Envie um e-mail com os detalhes do problema para `[sec@virtuaires.com.br]`. Caso não receba uma resposta, crie uma issue solicitando resposta sem mencionar a vulnerabilidade.

### O que incluir:
- Um resumo do problema de segurança detectado.
- Os passos para reproduzir a falha.
- O impacto potencial, considerando o modelo de execução local do Nanci.
- Se possível, sugestões ou indicação de onde o código pode estar falhando.
