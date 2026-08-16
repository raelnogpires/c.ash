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
- importação OFX e CSV, com detecção assistida de duplicatas;
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

O código ainda não foi inicializado. Quando isso acontecer, o fluxo padrão será:

```sh
go run ./cmd/cash
go build ./...
go test ./...
go vet ./...
```

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
