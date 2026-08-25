# Design e experiência

## Direção visual

O `[c]ash` combina sobriedade moderna com uma experiência acolhedora para
iniciantes. A interface deve parecer uma ferramenta pessoal refinada, não um
painel administrativo genérico nem uma planilha ornamentada.

A marca exibida é `[c]ash`; `c.ash` permanece como identificador técnico. A
interface usa linguagem cotidiana. Conceitos contábeis podem existir no domínio,
mas não são pré-requisito para entender uma tela.

Inter Variable é embutida no aplicativo e atende títulos, corpo e valores. A
hierarquia usa pesos e escala em vez de alternar entre famílias serifadas e sem
serifa; valores financeiros usam algarismos tabulares.

Lucide é a única família de ícones da interface. Setas, visibilidade, navegação,
privacidade e estados vazios usam glifos semânticos consistentes, sempre com
rótulo acessível quando fazem parte de um controle.

## Navegação e densidade

Uma barra lateral recolhível reúne:

- dashboard;
- transações;
- contas;
- cartões;
- orçamento;
- metas;
- configurações.

A densidade é intermediária: informações essenciais ficam visíveis, enquanto
detalhes avançados aparecem sob demanda. A aplicação funciona de uma largura
mínima documentada até tela cheia, sem precisar simular uma interface móvel.

Cada tela apresenta um único título principal. Cabeçalhos internos só aparecem
quando acrescentam contexto; o destino ativo não é repetido em eyebrow, card ou
rodapé.

## Temas

Existem três temas com a mesma estrutura e comportamento:

- **claro**;
- **escuro**;
- **gótico**, com preto, carvão, burgundy, bordas marcantes e ornamentação
  discreta.

Na primeira execução, o tema acompanha o sistema. A escolha pode ser alterada no
onboarding ou nas configurações. Tema muda tokens visuais, não posição, tamanho
ou significado dos componentes.

Verde comunica valores positivos; vermelho comunica despesas e alertas. Cor
nunca é o único sinal: rótulos, sinais, ícones e padrões visuais mantêm a
informação compreensível.

## Dashboard

A hierarquia principal começa pelo valor disponível para gastar. Saldo total,
valor reservado, receitas, despesas, próximas contas e metas fornecem contexto.

O MVP usa apenas gráficos que respondem a uma pergunta clara:

- receitas versus despesas por mês;
- gastos por categoria;
- evolução do saldo;
- progresso de orçamento e metas.

Gráficos devem oferecer alternativa textual ou semântica. Animação não pode
atrasar leitura, comparação ou interação com dados.

## Interação e movimento

Transições são curtas, funcionais e respeitam a preferência do sistema por
movimento reduzido. Movimento deve comunicar feedback, relação espacial ou
mudança de estado. Ações frequentes, navegação por teclado e dados em leitura
não recebem animação decorativa.

Estados de carregamento, vazio, sucesso, erro e indisponibilidade fazem parte de
cada fluxo. Mensagens explicam como corrigir o problema sem culpar o usuário.

## Acessibilidade

Acessibilidade é requisito de conclusão, não etapa posterior:

- contraste adequado nos três temas;
- navegação completa por teclado;
- ordem de foco previsível e foco visível;
- nomes e estados acessíveis em controles;
- escala de texto sem perda de conteúdo;
- informação nunca comunicada apenas por cor;
- suporte à preferência por movimento reduzido;
- conteúdo textual equivalente para gráficos.

Os três WebViews usados pelo Wails precisam de validação real. Semântica HTML é
o ponto de partida, não uma garantia automática de acessibilidade.

## Processo obrigatório de design

Antes de implementar uma tela principal:

1. Definir fluxo, objetivo e estados.
2. Produzir wireframe e hierarquia de informação.
3. Consultar shadcn MCP e 21st MCP para componentes e padrões adequados.
4. Aplicar as skills de Emil Kowalski para revisar acabamento e movimento.
5. Adaptar o resultado aos tokens e às regras do `[c]ash`.
6. Validar claro, escuro, gótico, teclado e movimento reduzido.
7. Versionar o código final no repositório.

As ferramentas auxiliam o desenvolvimento, mas não têm autoridade automática
sobre o produto. Código gerado deve ser compreendido, revisado e testado. O
aplicativo distribuído não depende dessas ferramentas.

## Aprovação de UI/UX

Uma tela do MVP só é aprovada quando:

- a hierarquia visual deixa a próxima ação evidente;
- uma despesa comum pode ser registrada em poucos segundos;
- a navegação é previsível;
- os três temas preservam legibilidade e identidade;
- teclado e movimento reduzido funcionam;
- componentes parecem parte do mesmo sistema;
- a interface não apresenta complexidade financeira antes de ser necessária.

Um dataset fictício separado cobrirá saldos positivos e negativos, cartão,
parcelas, orçamento e metas durante as revisões visuais.
