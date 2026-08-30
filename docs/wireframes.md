# Wireframes do sistema premium

Estes wireframes registram intenção, ordem de leitura e comportamento. Eles
partem da janela mínima de `960 × 640 px` e escalam para a referência de
`1180 × 760 px`. A geometria é a mesma nos temas claro, escuro e gótico; mudam
apenas superfícies, bordas, sombras e cor semântica.

## Shell e navegação agrupada

```text
┌────────────── 248 px ─────────────┬──────────────────────────────────────────┐
│ [c]ash                            │ Visão geral                 [⌄ ago/2026]│
│                                   │                              [+ Nova]    │
│ VISÃO                             │ Olá, Lia                                  │
│  ◉ Visão geral                    │                                          │
│                                   │ conteúdo da tela                         │
│ ATIVIDADE                         │                                          │
│  ↕ Movimentações                  │                                          │
│                                   │                                          │
│ PATRIMÔNIO                        │                                          │
│  ◫ Contas                         │                                          │
│  ◴ Cartões e faturas              │                                          │
│                                   │                                          │
│ PLANEJAMENTO                      │                                          │
│  ◷ Despesas fixas                 │                                          │
│  ◌ Orçamento                      │                                          │
│  ◎ Metas                          │                                          │
│                                   │                                          │
│ ORGANIZAÇÃO                       │                                          │
│  ⋮ Categorias                     │                                          │
│                                   │                                          │
│ SISTEMA                           │                                          │
│  ⚙ Configurações                 │ Lia · local                              │
└───────────────────────────────────┴──────────────────────────────────────────┘
```

No rail recolhido, cada destino conserva ícone, tooltip e nome acessível. Na
largura mínima, o conteúdo pode usar duas colunas somente quando houver espaço;
caso contrário, empilha. Em uma viewport menor que o mínimo de desktop, o rail
vira dock inferior e o menu guarda os destinos menos frequentes.

## Dashboard — cenário completo

```text
┌────────────────────────────── conteúdo ──────────────────────────────────────┐
│ Visão geral                                      agosto 2026  [olho] [+ Nova] │
│ Lia, este é o seu espaço                                                       │
│                                                                                │
│  Disponível com segurança                                                      │
│  R$ 4.733,00                         R$ 5.933,00 total   R$ 700,00 reservado  │
│  após despesas fixas                                                           │
│                                                                                │
│  ┌ Evolução do saldo ────────────┐  ┌ Por conta ───────────────────────────┐  │
│  │ R$ 5.933,00                    │  │ Conta principal        █████ 74,63 │  │
│  │       ╱╲___╱╲                  │  │ Reserva                 ██  1.850   │  │
│  │ fev. mar. abr. ... ago.        │  │ Carteira                ▏    120    │  │
│  └────────────────────────────────┘  └─────────────────────────────────────┘  │
│                                                                                │
│  Receitas  +R$ 5.200,00   Despesas  −R$ 3.273,20   Fatura  R$ 1.299,00         │
│                                                                                │
│  Atividade recente                                  Ver movimentações →        │
│  Salário de agosto          hoje              +R$ 5.200,00                     │
│  Mercado                    21 ago.            −R$ 64,20                       │
└──────────────────────────────────────────────────────────────────────────────┘
```

O stat disponível é a âncora. Valores ocultos mascaram somente saldos; sinais,
rótulos e alternativas textuais permanecem. O cenário negativo conserva a
mesma estrutura e troca a mensagem de apoio por uma orientação objetiva:
“Saldo abaixo de zero. Revise as próximas despesas ou mova recursos.”

## Dashboard — vazio e loading

```text
┌────────────────────────────── conteúdo ──────────────────────────────────────┐
│ Visão geral                                                   [+ Nova]         │
│                                                                                │
│                         [ícone de carteira]                                   │
│                         Seu espaço começa aqui                                │
│                  Adicione uma conta para ver seu saldo.                       │
│                         [+ Adicionar conta]                                   │
│                                                                                │
│  Atividade, orçamento e metas aparecem depois do primeiro registro.           │
└──────────────────────────────────────────────────────────────────────────────┘
```

Loading preserva título, stat e blocos com placeholders discretos. Erro mostra
causa curta, `Tentar novamente` e, quando possível, mantém o último conteúdo
válido em vez de trocar a página inteira por uma falha.

## Lançamento rápido — sheet

```text
┌────────────────────────────────── canvas ─────────────────┬──────── sheet ───┐
│ Movimentações                                               │ Nova movimentação│
│                                                             │ Registre em poucos│
│ contexto continua visível                                  │ passos            │
│                                                             │                  │
│                                                             │ [Despesa] Receita│
│                                                             │ Transferência     │
│                                                             │                  │
│                                                             │ O que foi?        │
│                                                             │ [                 ]│
│                                                             │ Valor             │
│                                                             │ [ R$              ]│
│                                                             │ Data              │
│                                                             │ [30/08/2026       ]│
│                                                             │ Conta             │
│                                                             │ [Conta principal ▾]│
│                                                             │ Categoria         │
│                                                             │ [Alimentação    ▾] │
│                                                             │                  │
│                                                             │ Mais detalhes  ⌄   │
│                                                             │                  │
│                                                             │ [Cancelar] [Salvar]│
└─────────────────────────────────────────────────────────────┴─────────────────┘
```

Descrição, valor, data, conta e categoria são o caminho comum. Parcelas, tags,
divisão e recorrência ficam em `Mais detalhes`. Transferência substitui
categoria por origem e destino:

```text
│ Nova transferência       │
│ [Despesa] [Receita]      │
│ [Transferência]          │
│ descrição · valor · data │
│ Origem [Conta principal] │
│ Destino [Reserva       ] │
│                 [Salvar]│
```

Ao salvar, a sheet fecha com feedback próximo da ação. Ao remover, a linha sai
da lista e aparece `Movimentação removida — Desfazer` por alguns segundos. O
foco volta ao acionador após fechar ou cancelar.

## Contas e cartões

```text
┌ Contas ───────────────────────────────────────────────────────────────────────┐
│ Patrimônio                                  [Buscar] [+ Adicionar conta]       │
│                                                                                │
│ Conta principal · Conta corrente                         R$ 7.463,00  ⋯       │
│ Reserva · Poupança                                       R$ 1.850,00  ⋯       │
│ Carteira · Dinheiro                                      R$   120,00  ⋯       │
│ Nubank · Cartão de crédito                               −R$ 3.500,00  ⋯      │
└──────────────────────────────────────────────────────────────────────────────┘
```

Editar uma conta ou ajustar saldo abre sheet; excluir abre modal de confirmação
quando há atividade. A área de cartões mostra limite disponível, fatura aberta,
data de vencimento e `Pagar fatura` em uma sheet de pagamento.

## Planejamento

```text
┌ Planejamento ─────────────────────────────────────────────────────────────────┐
│ Despesas fixas       Orçamento       Metas                                    │
│                                                                                │
│ PRÓXIMOS COMPROMISSOS                              [+ Adicionar despesa fixa]  │
│ Internet · vence 20 ago. · Conta principal                 R$ 99,00 [Confirmar]│
│ Aluguel · confirmado 10 ago.                               R$ 1.850,00        │
│                                                                                │
│ ORÇAMENTO DE AGOSTO                              [Editar orçamento]           │
│ R$ 5.100,00 de R$ 8.000,00                     ███████░░░ 63,75%               │
│                                                                                │
│ METAS                                                  [+ Nova meta]           │
│ Reserva de emergência        R$ 7.000 de R$ 15.000       █████░░░░ 46,67%      │
└──────────────────────────────────────────────────────────────────────────────┘
```

Confirmar ocorrência pergunta valor real e data em sheet. Orçamento e meta
mostram resumo primeiro; limites de categoria e alocações ficam em `Mais
detalhes` ou em sheet dedicada. Em cenário negativo, a barra e o texto deixam
claro o excedente; não depender apenas de vermelho.

## Configurações e estados de segurança

```text
┌ Configurações ───────────────────────────────────────────────────────────────┐
│ Aparência                                                                     │
│ Tema       ( Claro )  ( Escuro )  ( Gótico )                                  │
│                                                                                │
│ Privacidade                                                                   │
│ [ ] Ocultar saldos no dashboard                                               │
│                                                                                │
│ Dados locais                                                                  │
│ [Criar backup] [Restaurar backup] [Exportar dados]                            │
│                                                                                │
│ Segurança                                                                     │
│ Banco local protegido                                      [Ativar proteção]  │
└──────────────────────────────────────────────────────────────────────────────┘
```

Proteção, recuperação e restauração usam modal central com foco isolado.
Backup, exportação e atualização retornam sucesso, erro e progresso de forma
textual; nenhum estado depende de uma animação para ser entendido.

## Checklist de revisão

- [ ] Os grupos permanecem compreensíveis com rail expandido e recolhido.
- [ ] O primeiro viewport de `960 × 640` tem título, valor principal e próxima ação.
- [ ] A ação global abre sheet sem perder o contexto.
- [ ] Campos avançados ficam ocultos até serem necessários.
- [ ] Onboarding, completo, vazio e negativo são navegáveis no preview determinístico.
- [ ] Claro, escuro e gótico preservam contraste e ordem de foco.
- [ ] `prefers-reduced-motion` remove deslocamento sem remover feedback.
- [ ] Loading, erro, vazio e sucesso têm mensagens acionáveis.
