# Phase 1: Onboarding Conversacional no Telegram - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-10-03
**Phase:** 1-Onboarding Conversacional no Telegram
**Areas discussed:** Máquina de estados e persistência de sessão, Entrada de despesas fixas essenciais, Comportamento para usuários já cadastrados, Etapa opcional de investimentos, Estratégia de evicção de cache

---

## Máquina de Estados e Persistência de Sessão

| Option | Description | Selected |
|--------|-------------|----------|
| Em memória com Mutex no daemon do bot | Struct SessionStore com sync.RWMutex e mapa indexado por chat_id/telegram_id, sem novas tabelas | ✓ |
| Persistência em tabela no PostgreSQL | Tabela tb_bot_sessions no PostgreSQL acessada via endpoints HTTP na API | |

**User's choice:** Ficar em memória com o Mutex
**Notes:** O usuário optou pela simplicidade e ausência de overhead da memória local para o bot, complementada por uma estratégia de evicção de cache por TTL com ticker.

---

## Entrada de Despesas Fixas Essenciais

| Option | Description | Selected |
|--------|-------------|----------|
| Cadastro detalhado item a item | Usuário cadastra despesas linha a linha (ex: Aluguel 1500) gerando categorias na API | ✓ |
| Valor total consolidado | Usuário informa apenas uma soma aproximada dos custos fixos mensais | |

**User's choice:** Cadastro detalhado e cada item deve virar uma subcategoria de gastos fixos
**Notes:** A API botapi já possui suporte para mapear cada item da lista como categoria e despesa fixa associada ao usuário. Um botão inline de conclusão permite finalizar a lista com 1 clique.

---

## Comportamento para Usuários Já Cadastrados

| Option | Description | Selected |
|--------|-------------|----------|
| Exibir o Painel do Dia | Ao receber /start de usuário já cadastrado, buscar contexto e exibir S2S atual e menu | ✓ |
| Reiniciar o Onboarding | Resetar o setup e forçar o fluxo de configuração novamente | |

**User's choice:** Exibir o painel
**Notes:** Evita perda acidental de configurações anteriores e oferece feedback imediato do saldo do dia.

---

## Etapa Opcional de Investimentos

| Option | Description | Selected |
|--------|-------------|----------|
| Botões inline | Botões inline 'Adicionar Ativo' / 'Pular Etapa' | ✓ |
| Texto livre | Pergunta aberta instruindo a digitar 'pular' ou ativos | |

**User's choice:** Botões inline
**Notes:** Reduz atrito para novos usuários que ainda não investem e permite avançar com 1 toque.

---

## the agent's Discretion

- Formatação visual e emojis de cada mensagem no chat.
- Implementação detalhada do parser de strings de moedas e regex para ativos.
- Intervalo do ticker de limpeza de sessões (5 minutos) com TTL de 30 minutos.

---

## Deferred Ideas

Nenhuma ideia foi diferida; todas as discussões mantiveram-se estritamente alinhadas com o Milestone 2 do roadmap.
