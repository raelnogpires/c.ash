# Wireframes da primeira fatia vertical

Os wireframes abaixo registram fluxo, hierarquia e estados antes da implementação visual. As referências públicas de composição de diálogos do shadcn/ui, descoberta de componentes do 21st e princípios de acabamento/movimento de Emil Kowalski foram consideradas; os MCPs e a skill externa não estavam disponíveis no ambiente desta implementação.

## Onboarding

```text
┌──────────────────────────────┬────────────────────────────────────┐
│ [c]ash                       │ Configuração inicial · 1 de 1      │
│                              │ Vamos preparar seu espaço          │
│ Clareza começa com um        │ [ nome                             ]│
│ primeiro registro.           │ [ claro ] [ escuro ] [ gótico ]   │
│                              │                                    │
│ Privado por princípio        │ Sua primeira conta                 │
│ Dados somente no computador  │ [nome] [tipo] [saldo] [data]       │
│                              │ [Fazer depois] [Começar agora →]   │
└──────────────────────────────┴────────────────────────────────────┘
```

Estados: inicial, validação imediata, salvando, erro seguro. Pular cria apenas preferências e leva ao dashboard vazio.

## Estrutura principal e dashboard

```text
┌───────────────┬───────────────────────────────────────────────────┐
│ [c]ash        │ Hoje                         [+ Nova movimentação]│
│ Visão geral   │ Olá, nome                                        │
│ Contas        │ ┌───────────────────────────────────────────────┐ │
│ Movimentações │ │ Disponível após despesas fixas  R$ 0,00 [olho]│ │
│ Despesas fixas│ └───────────────────────────────────────────────┘ │
│ Aparência     │ [Evolução do saldo] [Saldo por conta]             │
│               │ [Receitas]       [Despesas]       [Resultado]    │
│ perfil/local  │ ┌ Atividade recente                            ┐ │
│               │ │ registros ou orientação de próximo passo     │ │
└───────────────┴───────────────────────────────────────────────────┘
```

Estados: carregando, falha ao abrir, sem conta, sem movimentação, saldo negativo e dados preenchidos.

O botão de olho persiste a preferência e mascara somente saldos do dashboard,
incluindo os gráficos. Receitas, despesas e valores das movimentações nunca são
mascarados por esse controle.

## Despesas fixas

```text
┌ Despesas fixas ─────────────────────────────────────────────────┐
│ Previsões pendentes   Confirmadas no mês   Regras ativas          │
│                                                                    │
│ O que falta pagar                                                 │
│ Internet · vence dia 20 · conta/categoria   R$ 99,90 [Confirmar] │
│                                                        [Dispensar]│
│                                                                    │
│ Seus compromissos                              [+ Adicionar]      │
│ Aluguel · dia 10 · conta/categoria            [Editar][Arquivar] │
└──────────────────────────────────────────────────────────────────┘
```

Confirmar abre um diálogo com valor real e data do pagamento. A regra é uma
previsão; só a confirmação cria a movimentação de despesa na conta escolhida.

## Contas e movimentação

```text
Contas: lista de cartões de saldo + formulário contextual de criação. O tipo é
escolhido entre três cartões explícitos: Conta corrente, Poupança e Dinheiro.

                      ┌ Novo registro ──────────────────┐
                      │ [Despesa][Receita][Transferência]│
                      │ descrição                       │
                      │ valor       data                │
                      │ conta       categoria pesquisável│
                      │              [Cancelar][Salvar] │
                      └─────────────────────────────────┘
```

O diálogo recebe foco no primeiro campo, fecha com Escape e apresenta tipo também por texto/sinal, não apenas por cor. Movimento é curto e removido com `prefers-reduced-motion`.

Conta e categoria usam combobox pesquisável. A busca ignora acentos e oferece
setas, Home, End, Enter, Escape e Tab, além de estados vazio, ocupado,
desabilitado e erro. Contas exibem tipo e saldo atual; categorias são filtradas
por Receita ou Despesa.

No modo Transferência, categoria dá lugar a origem e destino pesquisáveis:

```text
                      ┌ Nova transferência ────────────┐
                      │ descrição opcional             │
                      │ valor       data               │
                      │ origem      destino             │
                      │              [Cancelar][Salvar] │
                      └─────────────────────────────────┘
```

Cada linha mantém “Mais ações” visível. O menu oferece Editar e Remover;
transferências mostram `origem → destino`, seta própria e valor neutro. Editar
reabre o mesmo diálogo preenchido. Remover esconde a linha imediatamente e
mostra por oito segundos `Movimentação removida — Desfazer`; falhas restauram a
linha e apresentam mensagem segura. Ao fechar diálogo ou menu, o foco retorna
ao controle que iniciou a ação.
