# Especificação do produto

## Visão

O `[c]ash` é um aplicativo desktop de finanças pessoais para quem está
começando a se organizar. Ele deve substituir planilhas e dashboards separados
ao reunir registro, consulta, classificação e planejamento financeiro em uma
única fonte local.

O público inicial é uma única pessoa. O mercado inicial é o Brasil, com
português, real brasileiro, datas e formatos locais, além de conceitos comuns
como Pix, boleto e parcelamento. Textos e valores devem ser estruturados de
forma que uma futura internacionalização continue possível.

## Resultado esperado

O dashboard deve responder primeiro quanto o usuário pode gastar com segurança.
Para isso, ele apresenta também:

- saldo total;
- valor reservado e valor livre;
- receitas e despesas do mês;
- contas e faturas próximas;
- progresso de orçamento e metas.

O período padrão é o mês-calendário atual, com navegação entre meses e uma visão
anual secundária. O usuário aprovará o MVP principalmente pela UI/UX; a evolução
funcional será orientada pelos problemas recorrentes observados durante o uso
pessoal.

## Fluxos principais

### Primeiro acesso

Um assistente curto e pulável solicita nome, moeda, contas, saldos iniciais e
meta de reserva. O dashboard usa uma saudação como `Olá, {nome}`. Antes de
existirem transações, estados vazios orientam o próximo passo sem misturar dados
fictícios ao perfil real.

### Uso diário

Receitas e despesas podem ser adicionadas de qualquer tela por um botão global
ou atalho de teclado. O formulário começa compacto e revela campos avançados
sob demanda. Um rascunho incompleto permanece apenas durante a sessão e nunca
se torna uma movimentação sem confirmação.

Edições comuns são rápidas e oferecem desfazer. Confirmações ficam reservadas
para operações destrutivas ou difíceis de reverter. Exclusões vão para uma
lixeira recuperável, e alterações relevantes permanecem no histórico.

### Consulta

A busca textual pode ser combinada com filtros por período, conta, cartão,
categoria, etiqueta, tipo, status, faixa de valor e recorrência. O dashboard
pode ter blocos ocultados e reordenados, mas não será um construtor livre de
layouts.

## Alertas internos

O MVP alerta dentro do aplicativo quando houver:

- conta ou orçamento negativo;
- categoria próxima do limite;
- previsão próxima do vencimento;
- fatura próxima do fechamento ou vencimento;
- diferença de conciliação;
- backup atrasado.

Não haverá serviço em segundo plano nem notificação quando o aplicativo estiver
fechado.

## Privacidade, acesso e portabilidade

- O aplicativo funciona offline e não envia dados financeiros.
- Não há telemetria no projeto pessoal.
- Bloqueio por PIN ou senha é opcional e começa desativado.
- Criptografia em repouso é opcional e usa senha de segurança.
- Não existe chave-mestra nem recuperação remota da senha de criptografia.
- Uma chave de recuperação pode ser guardada pelo próprio usuário.
- Backups locais são automáticos e versionados, com pasta configurável.
- Exportações incluem CSV para transações e JSON para os dados estruturados.
- O modo portátil é opcional; a instalação normal usa o diretório de dados de
  cada sistema operacional.
- Duas instâncias não podem editar o mesmo banco simultaneamente.

Apagar todos os dados exige confirmação forte e oferta de backup imediatamente
antes da operação.

## Limites do MVP

Ficam explicitamente fora do primeiro lançamento:

- nuvem e sincronização;
- conexões automáticas com bancos;
- investimentos, ativos, criptomoedas e cotações;
- conversão entre moedas;
- módulo próprio de empréstimos e financiamentos;
- múltiplos perfis ou usuários;
- anexos de recibos e comprovantes;
- aplicativos móveis e interface web remota;
- telemetria e relatórios automáticos de erro;
- notificações fora do aplicativo.

Uma funcionalidade futura só entra no planejamento quando o uso pessoal revelar
um problema concreto e recorrente. O caso deve ser documentado antes da expansão
do escopo.

## Marcos e aprovação

O primeiro marco implementável cobre onboarding, criação de conta, lançamento
de transação, persistência SQLite e dashboard básico. A aprovação de UI/UX
considerará hierarquia clara, lançamento rápido, navegação previsível,
legibilidade nos três temas e uma identidade que não pareça um painel
administrativo genérico.

Depois do MVP, a qualidade funcional será avaliada pelo uso continuado:
conferência de saldos, ausência de dupla contagem, utilidade do dashboard e
vantagem prática sobre uma planilha.
