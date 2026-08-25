# [c]ash

Aplicativo desktop de controle financeiro pessoal para quem está começando a
organizar a própria vida financeira. O `[c]ash` será uma fonte única, local e
privada para registrar movimentações, acompanhar contas, planejar o mês e
construir uma reserva sem depender de planilhas ou de outro dashboard.

O projeto começa como uma ferramenta pessoal. Uma eventual transformação em
produto será decidida depois que o uso cotidiano validar a experiência e as
funcionalidades.

## Princípios

- **Local primeiro:** dados financeiros permanecem na máquina e o aplicativo
  funciona offline.
- **Feito para iniciantes:** linguagem cotidiana, bons padrões e orientação
  contextual, sem exigir conhecimento contábil.
- **Confiável:** valores não são contados duas vezes, alterações são
  rastreáveis e operações de gravação são atômicas.
- **Sem aprisionamento:** backup, restauração e exportação em formatos
  portáteis fazem parte do produto.
- **Paridade multiplataforma:** Windows, macOS e Linux terão os mesmos recursos
  no primeiro lançamento, com aparência consistente dentro das particularidades
  de cada sistema.
- **UI/UX como prioridade do MVP:** a interface deve ser clara, rápida,
  acessível e mais conveniente que uma planilha.

## Escopo do MVP

O primeiro lançamento incluirá:

- onboarding com nome, moeda, contas, saldos iniciais e meta de reserva;
- contas correntes, carteira, poupança e cartões de crédito com faturas;
- receitas, despesas, transferências, recorrências e parcelamentos;
- categorias, subcategorias, etiquetas e divisão de uma transação;
- orçamento mensal geral e limites opcionais por categoria;
- reserva de emergência e outras metas de economia;
- dashboard mensal, busca, filtros e alertas internos;
- importação OFX, CSV e extratos PDF do Itaú, Bradesco e Inter, com detecção de duplicatas;
- lixeira, histórico de alterações e ajustes de saldo explícitos;
- backup, restauração, exportação CSV/JSON e criptografia opcional;
- temas claro, escuro e gótico.

Não fazem parte do MVP: sincronização em nuvem, integração bancária automática,
investimentos e cotações, módulo especializado de empréstimos, múltiplos
usuários, anexos de comprovantes, aplicativos móveis, telemetria e notificações
com o aplicativo fechado.

## Stack planejada

- **Desktop:** Wails v2 estável
- **Domínio e persistência:** Go
- **Interface:** React, TypeScript e Vite
- **Armazenamento:** SQLite

As regras financeiras e a persistência ficam em Go. A interface cuida apenas
da apresentação, do estado transitório das telas e de validações imediatas de
formulário.

## Desenvolvimento

O primeiro fluxo vertical já está implementado com Go 1.25+, Wails v2.13.0,
React, TypeScript, Vite e SQLite puro em Go. Para validar as camadas a partir da
raiz do repositório:

```sh
go build ./...
go test ./... 
go vet ./...
cd frontend && npm install && npm test && npm run build
```

Para executar o aplicativo localmente com a ponte desktop embutida, a partir da
raiz do repositório:

```sh
go run -tags desktop,production,webkit2_41 ./cmd/cash
```

Em distribuições que ainda fornecem WebKitGTK 4.0, omita
`webkit2_41` da lista de tags. No Linux, o aplicativo também desativa por
padrão o renderizador DMA-BUF do WebKitGTK, que pode produzir uma janela branca
e sem interação em algumas combinações de GPU e driver.

Para trabalhar com recarregamento automático, instale o CLI fixado do Wails e
inicie pelo diretório que contém `wails.json`:

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0
cd cmd/cash
wails dev
```

Encerre a instância anterior com `Ctrl+C` antes de iniciar outra. O modo
`wails dev` usa as portas locais `34115` (ponte Wails) e `5173` ou a próxima
porta livre (Vite). Se o WebKitGTK encaminhar `runtime:ready` para a ponte de
navegador, use o comando `go run` acima; ele não depende dessa ponte de
desenvolvimento.

Para empacotar, use `wails build`. No Ubuntu 24.04 e em distribuições que usam
WebKitGTK 4.1, use `wails build -tags webkit2_41`. O binário é criado em
`build/bin/`. O tamanho mínimo da janela é 960 × 640.

## Docker

O contêiner produz e executa a versão Linux do aplicativo. Como o `[c]ash` é
uma aplicação desktop, ele usa o servidor X11 da máquina anfitriã para exibir a
janela; Docker não substitui o ambiente gráfico do sistema.

Em uma sessão Linux com X11, libere temporariamente o acesso ao display para o
usuário `root` do contêiner e inicie a aplicação:

```sh
xhost +si:localuser:root
docker compose up --build
```

Ao encerrar a aplicação, revogue a permissão concedida:

```sh
xhost -si:localuser:root
```

Os dados ficam no volume nomeado `cash-data`, separado do ciclo de vida do
contêiner. Para executar novamente, basta usar `docker compose up`; não é
necessário recriar o volume. A remoção explícita desse volume apaga os dados
locais do aplicativo:

```sh
docker compose down --volumes
```

Para validar as camadas Go em um ambiente Docker sem abrir a interface:

```sh
docker build --target test -t cash-test .
```

O uso da janela pelo contêiner requer Linux com X11. Em Wayland, use uma sessão
XWayland configurada pelo sistema ou execute o aplicativo localmente pelos
comandos da seção anterior.

## Atualizações do aplicativo

Versões instaladas consultam diariamente a última release estável do GitHub e
oferecem a atualização nas Configurações. A instalação sempre depende da
confirmação do usuário; o aplicativo baixa o pacote compilado da plataforma,
valida o SHA-256 fornecido pela API do GitHub e reinicia após a atualização.
Não há `git pull`, Go, Node, Wails ou Git na máquina de quem usa o aplicativo.

No Windows, a release entrega um instalador NSIS por usuário com o bootstrapper
do WebView2. Em Ubuntu, Linux Mint e Debian, a release entrega um `.deb` que
declara GTK3 e WebKitGTK; o sistema pode pedir a senha administrativa padrão,
mas não exige comandos manuais. Em macOS com Apple Silicon, a release entrega
um arquivo `.zip` com o aplicativo, disponível para download na página da
release do GitHub. Esse aplicativo não é assinado ou reconhecido pela Apple e
pode exigir que a pessoa usuária confirme sua abertura no macOS; ele também não
oferece atualização automática. Releases são geradas por tags semânticas como
`v0.2.0` no GitHub Actions.

Os dados de produção ficam em `c.ash/cash.db` sob o diretório de configuração
do usuário da plataforma. Testes injetam caminhos temporários e não usam dados
reais. Uma trava de arquivo impede duas instâncias de escreverem no mesmo banco.

## Importação de extratos PDF

Na tela de movimentações, o usuário pode associar à conta um extrato PDF de até
15 MB do Itaú, Bradesco ou Inter. O arquivo é lido localmente e não é preservado
depois do processamento. A gravação do lote é atômica.

Cada lançamento importado recebe uma origem automática e uma assinatura formada
pela conta, banco, data, natureza, valor, descrição e ocorrência. Isso permite
importar extratos todos os meses — inclusive com períodos sobrepostos — sem
duplicar o histórico. A assinatura permanece reservada quando uma movimentação é
editada ou excluída, portanto uma nova importação não desfaz a escolha do usuário.
Lançamentos automáticos anteriores à criação da conta local aparecem no histórico
e nos relatórios, mas não alteram novamente o saldo de abertura informado pelo
usuário.
PDFs protegidos por senha ou compostos apenas por imagens não são compatíveis.

Os comandos do frontend e de empacotamento serão documentados quando as
ferramentas forem configuradas.

## Documentação

- [Especificação do produto](docs/product.md)
- [Arquitetura](docs/architecture.md)
- [Modelo financeiro](docs/financial-model.md)
- [Design e experiência](docs/design.md)

Esses documentos registram o entendimento atual do MVP. Mudanças estruturais
devem atualizar a documentação correspondente junto com o código.

## Ordem de implementação

1. Arquitetura e modelo de dados.
2. Sistema visual e protótipo navegável.
3. Primeira fatia funcional: onboarding, conta, transação e dashboard.
4. Demais recursos financeiros do MVP.
5. Segurança, importação, backup e restauração.
6. Empacotamento e validação em Windows, macOS e Linux.

O primeiro marco será uma fatia vertical com dados reais no SQLite, não apenas
um protótipo visual. Até backup, restauração, criptografia e migrações estarem
validados, o desenvolvimento usará somente dados fictícios.
