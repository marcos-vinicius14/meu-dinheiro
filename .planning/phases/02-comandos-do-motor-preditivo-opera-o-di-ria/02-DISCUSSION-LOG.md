# Phase 2: Comandos do Motor Preditivo & Operação Diária - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-10-03
**Phase:** 2-Comandos do Motor Preditivo & Operação Diária
**Areas discussed:** Feedback e Apresentação do S2S e Gastos, Visualização da Simulação What-If, Mecânica do Check-in Diário, Interface de Acesso aos Comandos

---

## Feedback e Apresentação do S2S e Gastos

### Detalhamento visual do /s2s

| Option | Description | Selected |
|--------|-------------|----------|
| Card Completo de Ciclo | S2S diário (R$/dia), badge de saúde (🟢/🟡/🔴), dias restantes, saldo flexível restante e total já gasto no ciclo | ✓ |
| Card Ultra-Completo | Tudo do anterior + saldo real em conta corrente, total de despesas fixas e cobertura da reserva de emergência | |

**User's choice:** Card Completo de Ciclo
**Notes:** O usuário optou pelo equilíbrio visual com foco no ciclo atual e cota flexível sem sobrecarregar com dados de patrimônio.

### Recálculo do S2S no /gasto

| Option | Description | Selected |
|--------|-------------|----------|
| Comparativo Antes ➔ Depois | Exibe o gasto confirmado, S2S anterior vs novo S2S com variação diária (-R$ X/dia) e dias restantes | ✓ |
| Feedback Direto | Confirma o valor e categoria e exibe apenas o novo S2S atualizado sem histórico anterior | |

**User's choice:** Comparativo Antes ➔ Depois
**Notes:** Mostra claramente o impacto imediato da decisão de compra na cota diária até o final do ciclo.

### Envio de /gasto sem argumentos

| Option | Description | Selected |
|--------|-------------|----------|
| Explicar sintaxe com exemplos | Mostra como usar (/gasto 34.90 Almoço) e permite tentar novamente direto | ✓ |
| Entrar em modo guiado | Se enviado sem argumentos, o bot pergunta o valor e depois a descrição passo a passo | |

**User's choice:** Explicar sintaxe com exemplos

### Categorização da despesa rápida

| Option | Description | Selected |
|--------|-------------|----------|
| Categorização inteligente automática | A API associa pela descrição ou usa categoria flexível padrão | |
| Confirmar categoria via botões inline | O bot detecta o gasto e exibe botões com as categorias do usuário para escolha em 1 clique | ✓ |

**User's choice:** Confirmar categoria via botões inline

---

## Visualização da Simulação What-If

### Formato de projeção dos 12 ciclos

| Option | Description | Selected |
|--------|-------------|----------|
| Resumo Executivo + Pior Ciclo | Mostra impacto no ciclo atual, valor da parcela e destaca o ciclo mais crítico (menor S2S) dos 12 meses com alerta de risco | ✓ |
| Listagem Completa dos 12 Ciclos | Mostra tabela detalhada mês a mês com S2S projetado e status de todos os 12 ciclos futuros | |

**User's choice:** Resumo Executivo + Pior Ciclo
**Notes:** Evita mensagens gigantes de 12 linhas/tabelas; foca no ciclo que corre perigo de déficit.

### Sintaxe de parâmetros do /simular

| Option | Description | Selected |
|--------|-------------|----------|
| Sintaxe flexível com padrão à vista | /simular 1500 (à vista) ou /simular 1500 10 (10 parcelas de R$ 150,00) | ✓ |
| Exigir sempre os dois parâmetros | Obriga informar /simular <valor> <parcelas> mesmo para compras de 1 parcela | |

**User's choice:** Sintaxe flexível com padrão à vista

### Ação pós-simulação

| Option | Description | Selected |
|--------|-------------|----------|
| Botão inline "✅ Confirmar e Lançar" | Permite efetivar a compra simulada em 1 clique (como gasto flexível ou parcelamento real) | ✓ |
| Somente consulta (Read-only) | Simulação é puramente informativa; se comprar, o usuário lança separadamente | |

**User's choice:** Botão inline "✅ Confirmar e Lançar"

### Alerta de risco de déficit futuro

| Option | Description | Selected |
|--------|-------------|----------|
| Alerta explícito com diagnóstico | Avisa o mês exato do risco e sugere ajuste (ex: "Ciclo 3 ficará em Déficit: considere 15x ou adiar") | ✓ |
| Apenas badge visual | Indica o status de risco do pior mês sem sugestões textuais de ação | |

**User's choice:** Alerta explícito com diagnóstico

---

## Mecânica do Check-in Diário

### Estrutura do diálogo guiado

| Option | Description | Selected |
|--------|-------------|----------|
| Verificação de gastos pendentes | Pergunta se houve despesas não lançadas no dia com botões [Sim, lançar] e [Não, tudo certo] | ✓ |
| Fechamento direto sem perguntas | Assume que tudo já foi lançado e vai direto para a conciliação do dia | |

**User's choice:** Verificação de gastos pendentes

### Confirmação de saldo bancário

| Option | Description | Selected |
|--------|-------------|----------|
| Confirmação com ajuste opcional | Exibe saldo calculado e botões [✅ Sim, confere] ou [✏️ Ajustar saldo] caso haja divergência | ✓ |
| Não pedir saldo bancário | Apenas fecha o dia com base nas despesas registradas sem checar a conta corrente | |

**User's choice:** Confirmação com ajuste opcional

### Feedback final do check-in

| Option | Description | Selected |
|--------|-------------|----------|
| Relatório de Conquista Diária | Exibe gasto do dia vs meta, economia gerada que aumenta o S2S de amanhã e badge de saúde | ✓ |
| Confirmação Simples | Confirma o snapshot salvo e novo S2S sem cálculo de economia diária | |

**User's choice:** Relatório de Conquista Diária

### Execução repetida no mesmo dia

| Option | Description | Selected |
|--------|-------------|----------|
| Atualização idempotente (UPSERT) | Permite reexecutar o check-in caso surja um novo gasto tarde da noite, recalculando o snapshot do dia | ✓ |
| Bloquear repetição | Informa que o check-in de hoje já foi feito e pede para voltar amanhã | |

**User's choice:** Atualização idempotente (UPSERT)

---

## Interface de Acesso aos Comandos

### Disposição do teclado persistente (ReplyKeyboard)

| Option | Description | Selected |
|--------|-------------|----------|
| Grade 2x2 focada na operação diária | [💰 S2S Hoje] [💸 Lançar Gasto] / [🔮 Simular] [📝 Check-in] | ✓ |
| Grade 3x2 com suporte | [💰 S2S Hoje] [💸 Lançar Gasto] / [🔮 Simular] [📝 Check-in] / [❓ Ajuda] [🟢 Status] | |

**User's choice:** Grade 2x2 focada na operação diária

### Interação ao tocar em [💸 Lançar Gasto]

| Option | Description | Selected |
|--------|-------------|----------|
| Resposta com exemplo e atalho | Explica a sintaxe (/gasto 34.90 Almoço) para envio rápido na barra de digitação | ✓ |
| Entrar em fluxo interativo na FSM | Pede o valor e depois a descrição em perguntas separadas | |

**User's choice:** Resposta com exemplo e atalho

### Momento de ativação do teclado

| Option | Description | Selected |
|--------|-------------|----------|
| Automático no Onboarding e no /start | Fica permanentemente visível (ResizeKeyboard: true) na base do chat para usuários cadastrados | ✓ |
| Apenas sob demanda | Ativado somente se o usuário digitar /menu ou /teclado | |

**User's choice:** Automático no Onboarding e no /start

### Processamento de toques em [💰 S2S Hoje] e [📝 Check-in]

| Option | Description | Selected |
|--------|-------------|----------|
| Execução direta instantânea | [💰 S2S Hoje] roda o /s2s e [📝 Check-in] inicia o fluxo de check-in imediatamente | ✓ |
| Exibir confirmação/submenu antes de executar | Mostra opções antes de rodar o comando | |

**User's choice:** Execução direta instantânea

---

## the agent's Discretion

- Emojis, formatação e layout exato dos cartões de mensagem no Telegram.
- Estruturação dos novos endpoints internos na API Go (`POST /internal/transactions/simulations` e `POST /internal/transactions/checkin`).
- Nomenclatura dos estados internos da FSM para condução do diálogo guiado de check-in.

## Deferred Ideas

Nenhuma ideia diferida para fases futuras — escopo totalmente contido na Fase 2.
