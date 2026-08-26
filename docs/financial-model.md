# Modelo financeiro

## Conceitos fundamentais

O modelo é baseado em contas reais. Uma movimentação altera uma conta; uma
transferência move dinheiro entre contas próprias sem virar receita ou despesa.
Totais derivados são sempre recalculáveis a partir dos registros persistidos.

Esta fatia oferece quatro tipos de conta:

- conta corrente;
- carteira ou dinheiro;
- poupança.
- cartão de crédito.

Cartões são passivos: não entram no saldo disponível, mas o valor em aberto é
subtraído do patrimônio total. O pagamento reduz uma conta de dinheiro e a
dívida do cartão pelo mesmo valor, sem criar uma segunda despesa.

Conta corrente e carteira podem ficar negativas, com alerta. Cada conta começa
com um saldo inicial datado. Diferenças posteriores são corrigidas por um ajuste
de saldo explícito, com motivo e histórico, nunca pela edição silenciosa do
saldo calculado.

Poupança pode receber receitas, pagar despesas e participar de transferências,
mas nenhuma criação, edição, remoção ou restauração pode deixar seu saldo
calculado negativo. A operação inteira falha sem alterar o livro quando essa
invariante seria quebrada.

## Movimentações

Receitas e despesas usam a data em que ocorreram. Uma compra no cartão é despesa
na data da compra; o pagamento da fatura é transferência e não conta novamente
como gasto.

Uma movimentação pode conter:

- valor e data;
- descrição;
- estabelecimento ou pessoa;
- conta ou cartão;
- categoria e subcategoria;
- notas e etiquetas;
- partes vinculadas, quando dividida entre categorias;
- relação com recorrência, parcela, fatura, estorno ou importação.

As partes de uma transação dividida devem somar exatamente o valor total. Um
estorno é ligado à despesa original e reverte categoria, orçamento e fatura sem
inflar a renda.

Editar uma transação antiga recalcula saldos, orçamentos, faturas e metas
afetados. Alterações relevantes ficam no histórico. Exclusões são recuperáveis
pela lixeira.

Na representação atual, uma transferência é um registro atômico com conta de
origem e conta de destino distintas. A data deve ser igual ou posterior à
abertura das duas contas. Ela debita a origem, credita o destino pelo mesmo
valor e não altera receitas, despesas ou o saldo total. Transferências não têm
categoria; uma descrição vazia vira `Transferência para {destino}`.

Editar substitui os campos financeiros do mesmo registro e grava uma revisão.
Remover preenche `deleted_at`, também com revisão, e retira a movimentação de
saldos e indicadores. Restaurar limpa `deleted_at` somente após revalidar todas
as invariantes, inclusive o saldo de Poupança.

## Cartões e faturas

Cartões possuem limite, data de fechamento, vencimento, compras e faturas. Uma
compra parcelada gera todo o cronograma e mantém o vínculo com a compra original.
No fluxo atual, a compra inteira é reconhecida como despesa na data da compra;
as parcelas determinam somente em quais faturas o valor será cobrado. O dia de
fechamento já pertence ao ciclo seguinte. Valores são divididos igualmente e
qualquer diferença de centavos fica na primeira parcela.

Compras podem ser alteradas enquanto todas as faturas afetadas estiverem abertas
e sem pagamentos. Depois do fechamento ou de qualquer pagamento, o cronograma é
preservado. Uma fatura anterior opcional pode ser informada ao cadastrar o cartão
sem gerar uma despesa histórica.

Pagamentos totais e parciais são aceitos. Saldo não pago passa para a fatura
seguinte. O MVP não calcula automaticamente juros rotativos; juros cobrados são
registrados como despesa explícita.

## Recorrências e previsões

Rendas e gastos recorrentes geram previsões pendentes. O usuário confirma ou
ajusta a previsão antes que ela afete os valores realizados. Contas variáveis
podem sugerir o último valor ou uma média, sem lançamento automático.

O sistema distingue valores previstos e realizados em telas, filtros e totais.

### Despesas fixas mensais

Uma despesa fixa é uma regra recorrente com descrição, valor estimado, dia de
vencimento, conta de pagamento e categoria de despesa. A regra gera uma
ocorrência por mês, inclusive quando o vencimento é 29, 30 ou 31: nesses casos,
em meses mais curtos a data é o último dia do mês.

Ocorrências são previsões e não alteram o saldo realizado. Enquanto estiverem
pendentes e vencerem até o fim do mês corrente, são descontadas apenas do
**disponível após despesas fixas**. Ao confirmar, o usuário informa valor e
data reais; a aplicação cria uma despesa comum vinculada à ocorrência, marca a
previsão como confirmada e passa a alterar o saldo realizado uma única vez.
Uma ocorrência também pode ser dispensada e reaberta. Arquivar uma regra gera
eventuais meses pendentes até o próprio mês do arquivamento e preserva todo o
histórico. Se a regra for restaurada, o mês da restauração volta a ser elegível,
mas meses inteiramente arquivados não recebem ocorrências retroativas.

## Categorias e orçamento

O produto traz categorias iniciais editáveis. O usuário pode criar, renomear e
arquivar categorias, além de usar subcategorias opcionais.

Existe um orçamento geral por mês e limites opcionais por categoria. Saldo não
usado não passa ao mês seguinte por padrão, mas cada categoria pode habilitar
acúmulo. Meses permanecem editáveis. Um fechamento opcional alerta antes de
alterações retroativas, sem bloquear definitivamente o histórico.

## Reserva e metas

A reserva de emergência é uma meta de economia com valor-alvo e progresso. O
usuário também pode criar metas como viagem ou entrada de imóvel, com prazo
opcional.

Metas recebem alocações de uma ou mais contas. Alocar não cria uma conta, não
move dinheiro e não gera despesa; apenas separa parte do saldo para planejamento.
O dashboard distingue:

```text
saldo total = soma dos saldos das contas
valor reservado = soma das alocações válidas
valor livre = saldo elegível - valor reservado
```

O valor alocado não pode superar o saldo disponível nas contas associadas. A
regra exata para contas negativas e valores indisponíveis deverá ser expressa
como uma invariante testada durante a modelagem do domínio.

## Precisão e consistência

- Valores de BRL são inteiros em centavos.
- Operações não usam ponto flutuante.
- Transferências devem ser balanceadas.
- Uma compra no cartão e o pagamento da fatura nunca contam duas despesas.
- Ajustes, estornos e importações preservam sua origem.
- Nenhum total de dashboard é a única cópia de um dado financeiro.
- Reprocessar o histórico produz os mesmos saldos e indicadores.

Empréstimos especializados, investimentos, cotações e múltiplas moedas exigem
modelos próprios e permanecem fora do MVP.
