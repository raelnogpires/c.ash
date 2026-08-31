# Design e experiência

> Hallmark: modern-minimal · Workbench · designed-as-app · navegação N3 · sem
> footer · sem enriquecimento visual.

Este é o guia de implementação do sistema visual do `[c]ash`. O produto é uma
ferramenta pessoal de finanças, local-first, que deve transmitir precisão sem
parecer um painel de controle. O acabamento é minimalista e premium: pouca
decoração, bons intervalos, superfícies com profundidade discreta e movimento
que explica o que aconteceu.

## Arquitetura de informação

Os destinos são agrupados por intenção para que a pessoa encontre uma tarefa
antes de aprender o modelo de dados:

| Grupo | Destinos |
| --- | --- |
| Principal | Visão geral; Movimentações |
| Patrimônio | Contas; Cartões e faturas |
| Planejamento | Despesas fixas; Orçamento; Metas |
| Organização | Categorias |
| Configurações (rodapé) | Configurações |

No rail expandido, o grupo é um rótulo de seção e o destino ativo possui um
tratamento de seleção contínuo. No rail recolhido, os ícones permanecem
nomeados por tooltip e acessibilidade. Abaixo da largura confortável, o grupo
de maior frequência vira dock inferior e os destinos restantes continuam
disponíveis em menu. Configurações fica no rodapé do rail; o produto não tem
footer de conteúdo.

O cabeçalho do shell tem título único, contexto da tela, controles locais e
`+ Nova movimentação`. Ação e filtros não incluem seletor de período: navegação
mensal/anual está fora deste escopo. A ação global abre o lançamento rápido
sem navegar para longe do contexto atual.

## Contrato de layout

- Janela mínima: **960 × 640 px**.
- Tamanho de revisão: **1180 × 760 px**.
- Rail expandido: 248 px; recolhido: 80 px.
- Conteúdo fluido, com gutters generosos no desktop e empilhamento progressivo
  na largura mínima.
- Uma coluna de leitura confortável para listas e duas colunas apenas quando
  a relação entre os conteúdos for evidente.
- O primeiro viewport revela o título, o valor principal e a ação primária.

A hierarquia do dashboard é: disponível com segurança, saldo total/reservado,
entradas e saídas, faturas e previsões, depois atividade e planejamento. Uma
linha ou barra só deve existir se responder a uma pergunta; a mesma leitura
também aparece em texto para tecnologias assistivas.

## Material e tokens de superfície

O canvas, painéis, raised surfaces, sheets e modais formam cinco níveis
perceptíveis sem bordas pesadas. A separação vem de variação mínima de fundo,
hairline, sombra difusa e espaço. Use sombra para sugerir elevação, não para
decorar cada item.

Não usar gradientes de marketing, brilho, transparência espalhada, glassmorphism
ou uma grade de cards idênticos. O stat principal pode ter uma superfície
raised; listas e tabelas devem respirar como conteúdo, com divisores finos.

## Padrões de interação

### Sheets

Criar, editar e inspecionar contexto abre uma sheet pela direita no desktop
(aproximadamente 420–520 px). Em `960 × 640`, ela ocupa o espaço necessário e
pode tornar-se quase integral; em uma janela estreita, o padrão vira uma tela
empilhada com o mesmo cabeçalho.

Usar sheet para contas, cartões/faturas, ajustes de saldo, movimentações,
despesas fixas, ocorrências, orçamento, metas, alocações, categorias e filtros
avançados. A sheet tem título, descrição curta, campos em ordem de tarefa,
cancelamento previsível e ação primária no rodapé ou cabeçalho persistente.

### Modais

Usar modal central apenas para confirmação destrutiva, desbloqueio,
recuperação, restauração, atualização e decisões que bloqueiam o fluxo. O
scrim é discreto. Escape cancela ações reversíveis; o foco retorna sempre ao
elemento que abriu a camada.

### Formulários

O caminho comum de uma movimentação é descrição, valor, data, conta e
categoria. Receita, despesa e transferência mudam apenas o necessário:
transferência troca categoria por origem e destino. Parcelas, tags, divisão,
recorrência e outros detalhes ficam em `Mais detalhes` e aparecem sem apagar o
que já foi preenchido.

Erros aparecem junto ao campo e em um resumo anunciável. Campos inválidos não
dependem de vermelho; usam texto, ícone e foco. Comboboxes suportam pesquisa,
setas, Home/End, Enter, Escape e Tab.

## Movimento

Motion deve comunicar origem, hierarquia ou conclusão:

- 120 ms para pressionar, alternar e feedback mínimo;
- 180 ms para controles e menus;
- 260 ms para entrada/saída de página e sheet;
- 360 ms para feedback amplo;
- transform e opacity como propriedades principais;
- sem contagem animada de valores ou movimento ornamental.

Em `prefers-reduced-motion: reduce`, entradas são imediatas ou usam apenas
opacity mínima, blur animado é removido e feedback semântico permanece. O
layout não pode depender do fim de uma animação para ficar utilizável.

## Temas

Os temas são variações do mesmo sistema, não skins com outra arquitetura:

- **Claro:** papel quente, marfim e tinta verde profunda;
- **Escuro:** verde-preto, musgo elevado e acento menta;
- **Gótico:** carvão, regras vinho e rosa envelhecido, ornamentação discreta.

Todos mantêm a mesma geometria, ordem de foco, rótulos e sinalização. Positivo,
despesa, transferência e alerta recebem também sinal, texto ou ícone. Revisar
contraste, foco e estados disabled nos três temas.

## Estados de tela

Cada destino implementado deve ser revisado nestes estados:

| Estado | Requisito de conteúdo |
| --- | --- |
| Loading | Estrutura estável, indicador discreto e sem salto |
| Vazio | Motivo do vazio + próximo passo claro |
| Preenchido | Resumo, conteúdo e ação contextual |
| Negativo/alerta | Problema visível, causa curta e ação possível |
| Erro | Mensagem segura + tentar novamente ou alternativa |
| Sucesso | Confirmação próxima da mudança, sem ruído |
| Remoção | Feedback reversível quando a operação permitir |
| Indisponível | Dependência ou motivo explícito |

## Acessibilidade e revisão

O mínimo inclui headings únicos, landmarks coerentes, foco visível, teclado
completo, nomes acessíveis, contraste, zoom sem perda e conteúdo equivalente
para gráficos. Sheets e modais devem prender foco, fechar de forma previsível e
devolver foco ao acionador. Nenhum estado pode ser comunicado só pela cor.

## Preview local

O preview em `frontend/preview.html` é um entrypoint separado servido pelo
Vite. Ele monta o `App` de produção usando uma API mock local, determinística e
sem rede. `frontend/index.html` continua sem import do preview e segue sendo o
entrypoint de produção.

Use:

- `/preview.html?scenario=onboarding&theme=light` — configuração inicial;
- `/preview.html?scenario=rich&theme=light` — cenário completo;
- `/preview.html?scenario=empty&theme=dark` — primeiro passo sem registros;
- `/preview.html?scenario=negative&theme=gothic` — saldo negativo e orçamento
  excedido.

O controle no topo permite trocar cenário e tema sem recarregar manualmente;
`controls=0` o oculta durante capturas.
Após cada troca, validar navegação, sheets, formulários, estados, foco e
`prefers-reduced-motion` nas larguras `960 × 640` e `1180 × 760`.

## Definição de pronto

Uma tela está pronta quando a intenção é legível em poucos segundos, a tarefa
principal não compete com detalhes, a camada certa (canvas/panel/sheet/modal)
é usada, os três temas mantêm a mesma semântica e todos os estados essenciais
podem ser demonstrados no preview determinístico.
