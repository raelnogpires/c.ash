# Arquitetura

## Direção técnica

O `[c]ash` será construído com Wails v2 estável. O backend usa Go; a interface
usa React, TypeScript e Vite. A distribuição inclui o frontend compilado, sem
exigir Node.js na máquina do usuário.

Wails usa WebView2 no Windows, WebKit no macOS e WebKitGTK no Linux. Paridade
significa recursos e comportamento equivalentes, estrutura visual consistente e
acessibilidade validada. Pequenas diferenças de fonte, antialiasing e controles
nativos são aceitáveis; identidade pixel a pixel não é um requisito.

## Limites entre camadas

A arquitetura inicial será modular, sem microserviços ou abstrações antecipadas:

```text
React/TypeScript
    │ comandos e modelos de apresentação
    ▼
serviços de aplicação em Go
    │ casos de uso e transações
    ▼
domínio financeiro em Go
    │ regras e invariantes
    ▼
persistência SQLite/SQLCipher
```

- A UI não calcula saldos nem acessa SQLite diretamente.
- O domínio não depende de componentes visuais ou do runtime do Wails.
- Serviços de aplicação coordenam casos de uso, autorização local e transações.
- Repositórios tornam armazenamento e migrações explícitos e testáveis.
- TypeScript pode validar um formulário imediatamente, mas Go continua sendo a
  fonte da verdade.

## Organização planejada

```text
cmd/cash/             ponto de entrada do aplicativo
internal/domain/      entidades, valores e regras financeiras
internal/application/ casos de uso
internal/storage/     SQLite, migrações, backup e importadores
internal/platform/    integrações específicas de desktop
frontend/             React, TypeScript, componentes e temas
assets/               ícones e recursos estáticos
docs/                 especificações e decisões
```

Os nomes finais podem evoluir durante a inicialização, preservando a separação
de responsabilidades.

## Persistência e integridade

SQLite é a fonte principal. Dinheiro é armazenado como unidades inteiras da
menor fração da moeda; no MVP, centavos de BRL. Escritas que afetam mais de um
registro usam transações atômicas.

Transferências são uma única movimentação com `account_id` de origem e
`destination_account_id` de destino. O serviço valida as duas contas e simula
todos os saldos antes de confirmar a escrita; assim não existe uma metade da
transferência persistida. Consultas de saldo aplicam o débito na origem e o
crédito no destino, enquanto receitas e despesas mensais ignoram esse tipo.

Movimentações usam exclusão lógica por `deleted_at`. As consultas ativas,
saldos e dashboard omitem registros removidos, mas os dados permanecem no banco
para restauração. Cada criação, edição, remoção e restauração também grava uma
fotografia JSON em `transaction_revisions`. Esse histórico é de auditoria e
ainda não possui tela própria.

O banco possui uma versão de esquema independente da versão do aplicativo.
Migrações são automáticas, ordenadas e testadas. Antes de uma migração
irreversível, o aplicativo cria um backup. Versões novas abrem bancos antigos
por migração; não há promessa de que versões antigas abram bancos mais novos.

A migração `002_transactions` recria de forma transacional as restrições de
contas e movimentações, preserva as linhas existentes, acrescenta Poupança,
transferências, timestamps de atualização, exclusão lógica e revisões, e inclui
novas categorias sem substituir IDs ou rótulos já instalados.

A migração `003_fixed_expenses` adiciona a preferência local de ocultar saldos,
as regras de despesas fixas e suas ocorrências mensais. A confirmação de uma
ocorrência e a criação da movimentação vinculada acontecem na mesma transação
SQLite; um índice parcial impede duas movimentações ativas para a mesma
ocorrência.

Na inicialização, o aplicativo verifica bloqueio de instância e integridade do
banco. Bancos criptografados permanecem fechados até que a senha ou chave de
recuperação libere a chave do banco. O `Store` gerencia esse ciclo de vida:
operações comuns compartilham acesso, enquanto backup, conversão e restauração
pausam o acesso, trocam os arquivos e reabrem a mesma instância usada pelo
serviço de aplicação.

Conversão e restauração usam arquivos temporários no mesmo volume, `fsync`,
renomeação atômica, um journal `.operation.json` e cópias `.rollback`. Se o
processo for interrompido, a abertura seguinte restaura a versão anterior.
Antes de uma migração pendente em banco existente é criado um backup
`pre-migration`, fora da retenção automática.

## Datas e localização

Movimentações financeiras usam a data civil local como conceito principal.
Horários técnicos podem ser preservados para auditoria, mas conversões de fuso
não alteram o dia financeiro escolhido pelo usuário.

O MVP usa BRL e precisão de duas casas. Textos de interface não ficam embutidos
nas regras de domínio, preparando uma futura internacionalização sem ampliar o
escopo atual.

## Importação e exportação

- PDF usa extratores específicos para Itaú, Bradesco e Inter; arquivos
  protegidos por senha ou sem camada de texto são rejeitados.
- OFX 1.x SGML e OFX 2.x XML usam um importador genérico de movimentações. A
  instituição do arquivo não substitui o banco escolhido na interface.
- CSV detecta vírgula, ponto e vírgula ou tabulação e infere automaticamente
  cabeçalhos normalizados de data, descrição, valor, natureza, débito e crédito.
  O importador aceita UTF-8 e Windows-1252, datas brasileiras e ISO e notações
  decimais brasileiras e internacionais. Valores positivos sem sinal ou coluna
  de natureza são rejeitados quando sua direção é ambígua.
- PDF, OFX e CSV compartilham o limite de 15 MB e convergem para o mesmo lote
  atômico. A assinatura de conta, banco, data, natureza, valor, descrição e
  ocorrência evita duplicatas inclusive entre formatos, preservando a reserva
  da assinatura depois de edição ou exclusão pelo usuário.
- CSV usa UTF-8 com BOM e ponto e vírgula, contém somente movimentações ativas e
  inclui IDs, nomes de conta/categoria e origem de importação.
- JSON representa uma cópia estruturada completa e consistente, incluindo
  lixeira e revisões, sem chaves, preferências locais ou caminhos absolutos.
- `.cashbackup` é um ZIP versionado com `manifest.json`, `cash.db` e `cash.keys`
  quando aplicável. O manifesto registra versão, esquema, estado de criptografia
  e SHA-256 do banco. Exportações são relatórios, não entradas de restauração.
- O backup automático roda a cada sete dias e conserva as 12 versões automáticas
  mais recentes. Backups manuais e de segurança nunca são podados.

## Segurança

A criptografia integral é opcional e desativada por padrão. O driver fixado
`github.com/0xCarbon/go-sqlite3` v1.15.1 incorpora SQLCipher e recebe uma chave
bruta aleatória de 256 bits; a senha nunca é usada diretamente como chave do
banco. Argon2id deriva a chave de envelopamento com 64 MiB, 3 iterações e até 4
lanes. AES-256-GCM envolve a chave do banco, que só existe aberta na memória.

Uma chave de recuperação aleatória e independente recebe checksum, Base32 sem
padding e agrupamento para transcrição. `cash.keys` guarda apenas formato,
parâmetros do KDF, salts, nonces e envelopes. Não há chave-mestra, recuperação
remota ou keychain. A senha tem pelo menos 12 caracteres e é solicitada em toda
abertura. Troca e recuperação reescrevem o envelope; ativar/desativar converte o
banco com `sqlcipher_export` após um backup de segurança não podado.

SQLCipher depende de CGO, compilador C e OpenSSL/libcrypto. Builds e releases
declaram essa dependência e inspecionam os vínculos dos artefatos. O aviso BSD
completo está em [Licenças de código aberto](open-source-licenses.md) e nas
Configurações do aplicativo.

## Testes e entrega

A validação inclui:

- testes unitários do domínio e dos serviços em Go;
- testes de persistência, migração, importação, backup e restauração;
- testes de componentes críticos no frontend;
- testes da ponte Wails;
- fluxo ponta a ponta de onboarding, conta, transação, dashboard e backup;
- validação de teclado, contraste e movimento reduzido;
- verificação visual e funcional em Windows, macOS e Linux.

Uma versão não é lançada se uma plataforma suportada perder funcionalidade. A
integração contínua executará testes, `go vet`, builds de validação e verificações
do frontend. A matriz completa de empacotamento pode ficar restrita a versões
candidatas.

O projeto usa versionamento semântico a partir de `0.1.0`. Builds estáveis
consultam diariamente o GitHub Releases e exibem um aviso discreto; o download
e a instalação só ocorrem após confirmação explícita. O atualizador baixa
artefatos compilados, valida o digest SHA-256 publicado pelo GitHub e nunca usa
`git pull` ou ferramentas de desenvolvimento na máquina do usuário. Windows
usa um instalador NSIS por usuário; Ubuntu, Mint e Debian usam `.deb`, com os
runtimes GTK/WebKitGTK declarados como dependências do sistema.

## Dependências de design obrigatórias

Estas ferramentas são obrigatórias durante o desenvolvimento da interface:

- [Jpisnice/shadcn-ui-mcp-server](https://github.com/Jpisnice/shadcn-ui-mcp-server)
- [21st MCP](https://21st.dev/mcp), sucessor do
  [Magic MCP](https://github.com/21st-dev/magic-mcp)
- [emilkowalski/skills](https://github.com/emilkowalski/skills)

Elas apoiam descoberta, geração e revisão de componentes e interações. O código
resultante é adaptado, revisado e versionado no repositório. O aplicativo final
não chama MCPs, serviços de IA ou APIs de geração e não depende de chaves para
funcionar.

Bibliotecas incorporadas ao produto devem ter licença compatível, manutenção
ativa e propósito documentado. A licença do próprio `[c]ash` será decidida antes
de receber contribuições ou virar produto.
