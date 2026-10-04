# Phase 3: Gestão de Carteira de Ações no Telegram (`v0.1.0`) - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-10-04
**Phase:** 03-Gestão de Carteira de Ações no Telegram (`v0.1.0`)
**Areas discussed:** Formatação da Carteira, Sintaxe e Impacto do Comando /venda, Tratamento de Posição Zerada, Atalhos e Ações Rápidas, Suporte Pragmático a Renda Fixa

---

## Formatação da Carteira

| Option | Description | Selected |
|--------|-------------|----------|
| (Recomendado) Lista em Cards por Ativo | Cada ativo exibido com emojis e 2 linhas compactas (Ticker, Quantidade, PM e Total Investido), finalizando com o Total Geral da Carteira. | ✓ |
| Tabela Monoespaçada | Tabela em bloco de código (Markdown pre) com colunas alinhadas: TICKER \| QTD \| PM \| TOTAL. | |
| Resumo Sintético | Card compacto com Patrimônio Total Investido e botões inline para inspecionar os detalhes de cada ativo individualmente. | |

**User's choice:** Lista em Cards por Ativo
**Notes:** Evita quebras de linha desalinhadas em celulares.

| Option | Description | Selected |
|--------|-------------|----------|
| (Recomendado) Por Maior Posição Financeira (R$) | Ativos com maior valor total investido aparecem no topo, facilitando ver onde está a maior parte do patrimônio. | ✓ |
| Ordem Alfabética por Ticker (A-Z) | Organização previsível para encontrar rapidamente qualquer papel na lista. | |
| Por Último Aporte | Os ativos movimentados ou comprados mais recentemente aparecem no topo. | |

**User's choice:** Por Maior Posição Financeira (R$)
**Notes:** Destaque para onde o patrimônio do usuário está concentrado.

| Option | Description | Selected |
|--------|-------------|----------|
| (Recomendado) Completo | Exibir a % de alocação de cada ativo sobre a carteira e, no rodapé, destacar tanto o Total Investido quanto o Patrimônio Líquido Global (saldo bancário + investimentos). | ✓ |
| Focado apenas em Investimentos | Exibir % de alocação e Total Investido, sem misturar com o saldo da conta corrente. | |
| Minimalista | Apenas Ticker, Quantidade, PM e Total em R$ por ativo, sem cálculos de percentual de alocação. | |

**User's choice:** Completo
**Notes:** Dá visão unificada de liquidez e patrimônio investido.

| Option | Description | Selected |
|--------|-------------|----------|
| (Recomendado) Mensagem amigável com botão inline [➕ Adicionar Ativo] | Explica que a carteira está zerada, exibe exemplos práticos de uso e inclui botão inline que aciona orientações de aporte. | ✓ |
| Apenas texto orientador | Responde com mensagem direta explicando a sintaxe do comando /investimento. | |

**User's choice:** Mensagem amigável com botão inline [➕ Adicionar Ativo]

---

## Sintaxe e Impacto do Comando /venda

| Option | Description | Selected |
|--------|-------------|----------|
| (Recomendado) Sintaxe Flexível com Preço Opcional | Aceita tanto /venda PETR4 50 quanto /venda PETR4 50 a 41.50; quando o preço de venda for informado, calcula e exibe imediatamente o Lucro/Prejuízo em R$ e a rentabilidade % em relação ao PM. | ✓ |
| Preço Obrigatório | Sempre exigir o preço da venda (/venda <TICKER> <QTD> a <PREÇO>) para garantir apuração de lucro/prejuízo em todas as baixas. | |
| Apenas Quantidade | O comando só aceita /venda <TICKER> <QTD>, focando exclusivamente na redução do estoque/custódia do ativo sem cálculo de resultado. | |

**User's choice:** Sintaxe Flexível com Preço Opcional

| Option | Description | Selected |
|--------|-------------|----------|
| (Recomendado) Botão Inline Opcional de Crédito | Após registrar a venda, exibe botões inline [💳 Creditar R$ X no Saldo Líquido] e [🛡️ Manter Apenas na Carteira], dando controle se o resgate entra no fluxo de caixa diário (S2S). | ✓ |
| Apenas Registro na Carteira | A venda abate os ativos e calcula lucro/prejuízo, mas nunca altera o saldo bancário da conta corrente de forma automática. | |
| Crédito Automático no Saldo | Sempre que houver preço informado, o montante total liquidado é creditado diretamente como receita no saldo bancário do usuário. | |

**User's choice:** Botão Inline Opcional de Crédito

| Option | Description | Selected |
|--------|-------------|----------|
| (Recomendado) Rejeição com Atalho [Vender Todas as X] | Informa o erro de custódia insuficiente (ex: tentou vender 50, mas tem 30) e exibe um botão inline de 1 clique para vender o saldo total restante. | ✓ |
| Mensagem de Erro Simples | Apenas recusa a operação com mensagem explicativa informando a quantidade exata atualmente em custódia. | |

**User's choice:** Rejeição com Atalho [Vender Todas as X]

| Option | Description | Selected |
|--------|-------------|----------|
| (Recomendado) Card Completo de Liquidação | Mostra Ativo, Quantidade vendida, Preço praticado, Total apurado, Lucro/Prejuízo em R$ e %, e a Posição Remanescente (quantidade e valor total). | ✓ |
| Resumo Enxuto | Apenas confirma a baixa da quantidade e o novo saldo em custódia do ativo de forma concisa. | |

**User's choice:** Card Completo de Liquidação

---

## Tratamento de Posição Zerada

| Option | Description | Selected |
|--------|-------------|----------|
| (Recomendado) Execução Direta com Card de Encerramento | Conclui a venda imediatamente e destaca o fechamento do ciclo do ativo (ex: "🏁 Posição em PETR4 100% encerrada! Lucro total apurado: +R$ 450,00"). | ✓ |
| Confirmação Prévia via Botão Inline | Exibe aviso prévio avisando que a posição será extinta e pede confirmação antes de excluir o ativo da carteira: [✅ Confirmar Fechamento] [❌ Cancelar]. | |

**User's choice:** Execução Direta com Card de Encerramento

| Option | Description | Selected |
|--------|-------------|----------|
| (Recomendado) Botão Inline [↩️ Desfazer Venda] | Exibe botão inline efêmero na mensagem de confirmação para desfazer o lançamento caso o usuário tenha digitado quantidade ou ticker por engano. | ✓ |
| Sem Desfazer Automático | Não inclui botão de reversão; se o usuário digitou errado, basta readicionar o ativo via /investimento. | |

**User's choice:** Botão Inline [↩️ Desfazer Venda]

| Option | Description | Selected |
|--------|-------------|----------|
| (Recomendado) Notificação de Novo Ciclo | O novo aporte inicia um ciclo zerado (novo PM = preço da compra) e informa explicitamente que uma nova posição foi aberta para aquele ativo. | ✓ |
| Comportamento Silencioso | Registra normalmente o novo lote sem menção a ciclos anteriores, tratando como qualquer nova compra. | |

**User's choice:** Notificação de Novo Ciclo

| Option | Description | Selected |
|--------|-------------|----------|
| (Recomendado) Apenas Posições Ativas | A listagem da carteira (/carteira) exibe exclusivamente ativos com quantidade > 0, mantendo o relatório limpo e focado no patrimônio sob custódia. | ✓ |
| Botão Inline [📜 Posições Encerradas] | Exibe apenas os ativos ativos, mas adiciona um botão inline para consultar o histórico de papéis que já foram totalmente liquidados. | |

**User's choice:** Apenas Posições Ativas

---

## Atalhos e Ações Rápidas

| Option | Description | Selected |
|--------|-------------|----------|
| (Recomendado) Manter Teclado 2x2 Diário e Usar Comandos + Botões Inline | O teclado fixo segue focado no caixa diário (S2S, gasto, simulação, check-in); investimentos são operados via /carteira, /investimento e botões inline dentro das respostas. | ✓ |
| Expandir Teclado Persistente com [📈 Carteira] | Adicionar uma nova linha no teclado fixo do Telegram com o botão [📈 Carteira], totalizando 5 botões. | |

**User's choice:** Manter Teclado 2x2 Diário e Usar Comandos + Botões Inline

| Option | Description | Selected |
|--------|-------------|----------|
| (Recomendado) [➕ Novo Aporte] e [🔄 Atualizar] | Botões no rodapé da carteira para orientar novo aporte de ativos e permitir atualizar as cotações/posições no mesmo card sem duplicar mensagens. | ✓ |
| Sem botões inline | Relatório puramente textual, sem botões anexados à mensagem. | |

**User's choice:** [➕ Novo Aporte] e [🔄 Atualizar]

| Option | Description | Selected |
|--------|-------------|----------|
| (Recomendado) Suporte a Sinônimos e Aliases Naturais | Aceitar /carteira, /investimentos e /portfolio para visualização; /investimento ou /aporte para compra; e /venda ou /vender para baixa. | ✓ |
| Estritamente Canônicos | Aceitar apenas /carteira, /investimentos, /investimento e /venda sem aliases adicionais. | |

**User's choice:** Suporte a Sinônimos e Aliases Naturais

| Option | Description | Selected |
|--------|-------------|----------|
| (Recomendado) Mensagem Educativa com Exemplos Copiáveis | Segue o padrão do /gasto da Fase 2: responde na hora com exemplos claros e formatados para fácil cópia e colagem no chat. | ✓ |
| Diálogo Conversacional FSM Passo a Passo | Abre um fluxo guiado perguntando um dado por vez: Ticker -> Quantidade -> Preço. | |

**User's choice:** Mensagem Educativa com Exemplos Copiáveis

---

## Suporte Pragmático a Renda Fixa

| Option | Description | Selected |
|--------|-------------|----------|
| (Recomendado) Suporte Pragmático via Tickers na Fase 3 | Permitir cadastrar títulos de renda fixa como ativos normais na custódia atual (ex: TD-SELIC, CDB-INTER, com quantidade e valor unitário), sem alterar o banco de dados nem criar indexadores. | ✓ |
| Criar Nova Fase no Roadmap (Renda Fixa Completa) | Registrar uma fase futura dedicada para Renda Fixa com modelagem avançada (indexador % CDI/IPCA, data de vencimento, liquidez e IR regressivo), mantendo a Fase 3 focada em ativos cotados. | |

**User's choice:** Suporte Pragmático via Tickers na Fase 3
**Notes:** O usuário pontuou a necessidade de incluir Renda Fixa. Optou-se pela solução pragmática e elegante de suportar tickers flexíveis de até 12 caracteres (ex: `TD-SELIC`, `CDB-INTER`), permitindo gerenciar títulos públicos e bancários na mesma estrutura determinística de quantidade e preço médio.

---

## the agent's Discretion

- Emojis e formatação Markdown fina dos cards no Telegram.
- Estruturação dos DTOs dos novos endpoints da API (`GET /internal/investments` e `POST /internal/investments/sell`).
- Implementação interna da reversão de venda no handler de callback (`[↩️ Desfazer Venda]`).

## Deferred Ideas

- **Módulo Avançado de Renda Fixa:** Modelagem avançada de ativos de renda fixa contendo indexadores dinâmicos (% CDI, IPCA + taxa pré-fixada), datas de vencimento, liquidez, marcação a mercado e cálculo de IR regressivo e come-cotas.
