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
persistência SQLite
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

O banco possui uma versão de esquema independente da versão do aplicativo.
Migrações são automáticas, ordenadas e testadas. Antes de uma migração
irreversível, o aplicativo cria um backup. Versões novas abrem bancos antigos
por migração; não há promessa de que versões antigas abram bancos mais novos.

Na inicialização, o aplicativo verifica bloqueio de instância e integridade do
banco. Falhas nunca devem deixar uma operação parcialmente aplicada.

## Datas e localização

Movimentações financeiras usam a data civil local como conceito principal.
Horários técnicos podem ser preservados para auditoria, mas conversões de fuso
não alteram o dia financeiro escolhido pelo usuário.

O MVP usa BRL e precisão de duas casas. Textos de interface não ficam embutidos
nas regras de domínio, preparando uma futura internacionalização sem ampliar o
escopo atual.

## Importação e exportação

- OFX tem importador genérico.
- CSV usa um assistente para mapear colunas e salvar modelos por banco.
- Uma segunda importação apresenta possíveis duplicatas para confirmação; não
  descarta transações silenciosamente.
- CSV é o formato de exportação tabular.
- JSON representa uma cópia estruturada completa.
- O backup integral preserva banco, versão e metadados necessários à restauração.

## Segurança

A criptografia do banco é opcional e protegida por uma senha conhecida apenas
pelo usuário. Não haverá chave-mestra ou recuperação remota. Segredos nunca são
gravados no repositório, nos logs ou em arquivos exportados sem proteção.

Bloqueio de interface e criptografia são conceitos separados: um PIN pode
impedir acesso casual, enquanto a senha de criptografia protege os dados em
repouso.

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

O projeto usa versionamento semântico a partir de `0.1.0`. Atualizações não são
silenciosas: no projeto pessoal, a verificação será manual ou um aviso discreto.
Cada sistema receberá empacotamento apropriado quando a distribuição for
implementada.

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
