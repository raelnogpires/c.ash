# Sistema visual — [c]ash

> Hallmark: modern-minimal · Workbench · designed-as-app · navegação N3 · sem
> footer · sem enriquecimento visual.

Este documento é a referência de produto para o redesign do `[c]ash`: uma
ferramenta pessoal de finanças, local-first, silenciosa e precisa. A direção é
minimalista e premium, com a clareza espacial de um utilitário de plataforma,
sem transformar o app em uma vitrine ou em um painel administrativo.

## Princípios

1. **Uma pergunta por tela.** O título, o primeiro número e a primeira ação
   devem explicar o que a pessoa pode fazer agora.
2. **Hierarquia antes de decoração.** Espaço, tipografia e alinhamento carregam
   a maior parte da interface; material, borda e sombra só separam camadas.
3. **Detalhe sob demanda.** O caminho comum é curto. Campos, filtros e
   explicações avançadas aparecem quando são necessários.
4. **Dados honestos.** Saldos negativos, vazios, carregamento e falhas têm
   tratamento explícito. Nenhum gráfico ou animação mascara o estado real.
5. **Privacidade por padrão.** Dados financeiros permanecem no computador e
   preferências de visibilidade são comunicadas com clareza.

## Arquitetura de informação

A navegação deixa de ser uma lista plana e passa a ser organizada por intenção.
Os grupos são visíveis quando a barra está expandida e continuam acessíveis
quando ela está recolhida.

| Grupo | Destinos | Pergunta respondida |
| --- | --- | --- |
| **Principal** | Visão geral; Movimentações | Como está meu dinheiro e o que mudou? |
| **Patrimônio** | Contas; Cartões e faturas | Onde está o dinheiro e o que vence? |
| **Planejamento** | Despesas fixas; Orçamento; Metas | O que preciso prever e priorizar? |
| **Organização** | Categorias | Como quero classificar meus registros? |
| **Configurações (rodapé)** | Configurações | Como ajusto privacidade, dados e aparência? |

O rail mostra `[c]ash`, os grupos Principal, Patrimônio, Planejamento e
Organização, o destino ativo e o perfil local. Configurações fica no rodapé do
rail; não há footer de conteúdo. O estado
recolhido mantém ícone, `aria-label`, tooltip e foco visível; nunca depende só
de cor. Em larguras estreitas, o mesmo conjunto vira um dock inferior com os
destinos de maior frequência e um menu para os demais.

## Contrato de janela e layout

- **Mínimo suportado:** `960 × 640 px`.
- **Referência de revisão:** `1180 × 760 px`.
- **Rail:** 248 px expandido, 80 px recolhido.
- **Conteúdo:** coluna fluida, largura máxima de leitura de aproximadamente
  1248 px, com gutters que diminuem gradualmente até a largura mínima.
- **Cabeçalho:** título e contexto à esquerda; filtros locais e ação global à
  direita. Navegação mensal/anual não faz parte deste escopo.
- **Ação global:** `+ Nova movimentação` fica no cabeçalho do shell e abre uma
  sheet de lançamento rápido sem apagar o contexto da tela.

O primeiro viewport deve revelar título, valor principal, ação primária e o
início do conteúdo da tela. Listas longas rolam dentro do conteúdo; rail,
sheet e cabeçalho não saltam durante a navegação.

## Material e camadas

O material é sutil: superfícies opacas, contraste baixo e separadores finos.
Transparência e blur ficam reservados para a camada que está sobre o app, nunca
para cada card.

| Camada | Uso | Tratamento |
| --- | --- | --- |
| **Canvas** | fundo da janela | cor de papel/carvão, sem gradiente obrigatório |
| **Panel** | seção e lista | superfície levemente elevada, hairline e raio moderado |
| **Raised** | stat principal, menu, popover | sombra difusa curta e contraste de superfície |
| **Sheet** | edição e criação contextual | painel sólido, borda lateral e sombra de profundidade |
| **Modal** | confirmação ou bloqueio | scrim discreto, foco isolado, largura curta |

Cards não devem formar uma grade de caixas iguais. O dashboard usa um stat
dominante, seções abertas e divisores; cartões só aparecem quando agrupam uma
decisão ou uma leitura relacionada. Não usar brilho, gradiente decorativo,
neumorfismo ou glassmorphism como linguagem principal.

## Sheets, modais e formulários

Sheets são o padrão para criação, edição e detalhe que preserva o contexto:

- conta e ajuste de saldo;
- cartão e pagamento de fatura;
- despesa fixa e confirmação de ocorrência;
- orçamento, meta e alocação;
- categoria;
- filtros avançados de movimentações;
- nova movimentação rápida.

No desktop, a sheet entra pela direita com aproximadamente 420–520 px. Em
`960 × 640`, ela pode ocupar quase toda a largura sem esconder o cabeçalho do
próprio formulário. Em uma janela estreita, vira uma tela empilhada com o
mesmo cabeçalho, ação e ordem de foco.

Modal central fica restrito a confirmação destrutiva, desbloqueio, recuperação,
restauração, atualização e mensagens que exigem decisão imediata. Toda camada
tem título, descrição quando necessário, ação primária nomeada e cancelamento
óbvio. Escape fecha apenas o que é reversível; o foco retorna ao acionador.

Formulários mostram primeiro descrição, valor, data e conta. Categoria, origem,
destino e recorrência aparecem de acordo com o tipo. `Mais detalhes` revela
parcelas, tags, divisão, observações e regras avançadas; o estado aberto não é
perdido ao validar. Erros ficam junto ao campo e também em um resumo acessível.

## Tipografia, ícones e números

- Geist Variable embutida é usada no display e no wordmark; Inter Variable fica
  no corpo, controles e valores financeiros.
- Títulos são romanos, com peso e espaço para criar hierarquia; a dupla é
  funcional, não ornamental.
- Valores financeiros usam algarismos tabulares, alinhamento consistente e
  sinal textual (`+`, `−`) além de cor.
- Lucide é a família única de ícones, com traço consistente e nome acessível em
  qualquer controle sem texto.
- O texto de interface é direto, em português brasileiro; rótulos não devem
  depender de jargão contábil.

O dashboard responde primeiro “quanto posso usar com segurança?”. Saldo
disponível é o maior stat; saldo total, reservado, receitas, despesas,
faturas, orçamento e metas aparecem como contexto progressivo. Gráficos só
entram quando respondem uma pergunta identificável e sempre têm resumo textual.

## Temas

Os três temas compartilham geometria, conteúdo, ordem de foco e significado:

- **Claro:** papel quente, superfícies marfim, tinta verde profunda e acento
  mineral.
- **Escuro:** papel verde-preto, superfícies de musgo e acento menta.
- **Gótico:** carvão, regras vinho e acento rosa envelhecido, com ornamentação
  mínima e sem textura pesada.

Tema altera tokens de cor, borda, sombra e scrim; nunca altera o layout nem
esconde informação. Verde indica positivo, vermelho indica despesa/alerta e
neutro indica transferência. Cada significado também tem rótulo, sinal ou
ícone. O contraste e o foco devem ser revisados nos três temas.

## Movimento e preferência do sistema

Motion é funcional: orienta a origem de uma sheet, confirma uma mudança ou
mostra o resultado de uma ação reversível.

- pressão e microinteração: `120 ms`;
- controles e menus: `180 ms`;
- páginas e sheets: `260 ms`;
- feedback amplo: `360 ms`;
- easing único de saída suave, com transform e opacity como propriedades
  preferenciais;
- sem contagem animada de dinheiro, parallax ou movimento ornamental;
- feedback permanece compreensível quando a animação é interrompida.

Com `prefers-reduced-motion: reduce`, remover deslocamento e blur animado,
reduzir transições a uma troca imediata de estado e preservar foco, anúncio e
ordem de conteúdo. O preview deve ser revisado tanto no modo normal quanto no
reduzido.

## Estados e acessibilidade

Cada destino precisa de uma representação para:

- carregando, com estrutura estável e indicador discreto;
- vazio, explicando por que está vazio e oferecendo o próximo passo;
- preenchido, com ações e resumo;
- negativo/alerta, sem esconder o problema nem usar tom punitivo;
- erro recuperável, com causa curta e ação de tentar novamente;
- sucesso e remoção reversível, com confirmação próxima da ação;
- indisponível/desabilitado, explicando a dependência.

Teclado, foco visível, nomes acessíveis, headings únicos, contraste, zoom e
leitores de tela fazem parte da definição de pronto. Tabelas, barras e gráficos
precisam de alternativa textual. Sheet e modal mantêm foco dentro da camada,
fecham de forma previsível e não deixam o foco perdido no canvas.

## Preview de revisão

O entrypoint de revisão é `frontend/preview.html`, servido pelo Vite em
`/preview.html`. Ele monta o `App` real com uma API mock local e determinística;
`frontend/index.html` continua sendo o entrypoint de produção e não importa o
preview.

Controles no canto superior permitem alternar:

- `Onboarding` (`scenario=onboarding`): configuração inicial sem nenhum dado;
- `Completo` (`scenario=rich`): contas, cartão, atividade, orçamento, metas e
  atualização disponível;
- `Vazio` (`scenario=empty`): dashboard configurado ainda sem contas ou registros;
- `Atenção` (`scenario=negative`): saldo negativo e orçamento excedido;
- `Claro`, `Escuro` e `Gótico` (`theme=light|dark|gothic`).

Exemplos: `/preview.html?scenario=rich&theme=dark` e
`/preview.html?scenario=negative&theme=gothic`. Use `controls=0` para capturas
sem o painel de revisão. A API mock cobre os fluxos
visíveis para permitir navegação manual, sem persistência e sem chamadas de
rede. O preview é uma ferramenta de inspeção, não uma nova rota de produto.

## Critérios de revisão

Uma entrega deste sistema está pronta quando, em `960 × 640` e `1180 × 760`:

1. a arquitetura agrupada é compreensível em estado expandido e recolhido;
2. o stat principal e a próxima ação são encontrados sem caça visual;
3. criar ou editar um registro usa uma sheet, preserva contexto e devolve foco;
4. vazio, negativo, loading e erro têm conteúdo útil;
5. claro, escuro e gótico conservam legibilidade e semântica;
6. motion reduzido remove deslocamento sem remover feedback;
7. teclado, leitor de tela e zoom não quebram a tarefa;
8. o preview percorre os fluxos com dados determinísticos e não altera o
   entrypoint de produção.
