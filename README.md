# [c]ash

Aplicativo desktop de controle financeiro pessoal para quem está começando a
organizar a própria vida financeira. O `[c]ash` será uma fonte única, local e
privada para registrar movimentações, acompanhar contas, planejar o mês e
construir uma reserva sem depender de planilhas ou de outro dashboard.

O projeto começa como uma ferramenta pessoal. Uma eventual transformação em
produto será decidida depois que o uso cotidiano validar a experiência e as
funcionalidades.

> MVP implementado. A distribuição oficial é feita pela página de
> [releases no GitHub](https://github.com/raelnogpires/c.ash/releases).

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

## Recursos

O MVP oferece:

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

## Stack

- **Desktop:** Wails v2 estável
- **Domínio e persistência:** Go
- **Interface:** React, TypeScript e Vite
- **Armazenamento:** SQLite/SQLCipher (criptografia opcional)

As regras financeiras e a persistência ficam em Go. A interface cuida apenas
da apresentação, do estado transitório das telas e de validações imediatas de
formulário.

## Desenvolvimento e validação

O primeiro fluxo vertical já está implementado com Go 1.25+, Wails v2.13.0,
React, TypeScript, Vite e SQLite/SQLCipher. A compilação exige CGO, compilador C
e headers do OpenSSL (`libssl-dev` em Debian/Ubuntu, `openssl@3` no macOS ou o
pacote MinGW equivalente no Windows). Para validar as camadas a partir da raiz:

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
oferece atualização automática. Releases são geradas automaticamente pelo
GitHub Actions quando uma tag semântica como `v0.2.0` é publicada. O fluxo
valida a tag, executa os testes e gera os artefatos abaixo:

- `cash_<versão>_windows_amd64_setup.exe`: instalador NSIS para Windows 10/11
  em computadores Intel ou AMD;
- `cash_<versão>_linux_amd64.deb`: pacote Debian para Linux amd64;
- `cash_<versão>_darwin_arm64.zip`: aplicativo para macOS com Apple Silicon.

Para publicar uma versão, crie e envie uma tag no formato `vMAJOR.MINOR.PATCH`:

```sh
git tag v0.2.0
git push origin v0.2.0
```

Depois que o workflow terminar, os instaladores estarão disponíveis na página
da release correspondente.

Os dados de produção ficam em `c.ash/cash.db` sob o diretório de configuração
do usuário da plataforma. Testes injetam caminhos temporários e não usam dados
reais. Uma trava de arquivo impede duas instâncias de escreverem no mesmo banco.

Backups automáticos ficam por padrão em `c.ash/backups` sob o diretório de
configuração, rodam a cada sete dias e retêm 12 versões automáticas. Arquivos
`.cashbackup` manuais e de segurança não são removidos pela retenção. A pasta
escolhida fica em `backup-settings.json`, fora do banco financeiro. Quando a
criptografia opcional está ativa, `cash.keys` contém somente envelopes
criptográficos; senha e chave de recuperação nunca são armazenadas.

## Importação de extratos PDF, OFX e CSV

Na tela de movimentações, o usuário pode associar à conta um extrato PDF, OFX ou
CSV de até 15 MB e indicar Itaú, Bradesco ou Inter como origem. O arquivo é lido
localmente e não é preservado depois do processamento. A gravação do lote é
atômica. PDFs usam os layouts específicos dos três bancos; OFX 1.x e 2.x são
lidos de forma genérica, sem substituir o banco selecionado pelos metadados do
arquivo.

No CSV, o aplicativo reconhece automaticamente delimitadores por vírgula,
ponto e vírgula ou tabulação e cabeçalhos comuns de data, descrição, valor,
tipo, débito e crédito. São aceitas datas brasileiras e ISO, valores brasileiros
e internacionais, colunas com sinal, valores acompanhados de natureza e colunas
separadas de débito/crédito. CSV em UTF-8 ou Windows-1252 é compatível; valores
sem sinal ou natureza suficiente são rejeitados para evitar inverter receitas e
despesas.

Cada lançamento importado recebe uma origem automática e uma assinatura formada
pela conta, banco, data, natureza, valor, descrição e ocorrência. Isso permite
importar extratos todos os meses — inclusive com períodos sobrepostos — sem
duplicar o histórico. A assinatura permanece reservada quando uma movimentação é
editada ou excluída, portanto uma nova importação não desfaz a escolha do usuário.
Lançamentos automáticos anteriores à criação da conta local aparecem no histórico
e nos relatórios, mas não alteram novamente o saldo de abertura informado pelo
usuário.
PDFs protegidos por senha ou compostos apenas por imagens não são compatíveis.

## Documentação

- [Especificação do produto](docs/product.md)
- [Arquitetura](docs/architecture.md)
- [Modelo financeiro](docs/financial-model.md)
- [Design e experiência](docs/design.md)
- [Licenças de código aberto](docs/open-source-licenses.md)

Esses documentos registram o entendimento atual do MVP. Mudanças estruturais
devem atualizar a documentação correspondente junto com o código.

## Estado do projeto

O fluxo principal do MVP está implementado e coberto por testes de domínio,
persistência, importação, segurança e interface. O empacotamento de release é
validado em GitHub Actions para Windows, Linux e macOS Apple Silicon.
