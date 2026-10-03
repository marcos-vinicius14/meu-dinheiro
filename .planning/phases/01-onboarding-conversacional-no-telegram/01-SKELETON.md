# Walking Skeleton — Meu Dinheiro (Onboarding Conversacional Telegram)

**Phase:** 1
**Generated:** 2026-10-03

## Capability Proven End-to-End

Um usuário novo envia `/start` no Telegram, é guiado por uma máquina de estados finitos interativa com botões inline que coleta saldo inicial, ciclo financeiro, despesas essenciais detalhadas, meta de reserva 6x/12x e investimentos opcionais, salvando o setup via `POST /internal/users/onboarding` e recebendo o primeiro relatório com Saldo Seguro Diário (S2S) e indicador de saúde do ciclo; ou, sendo usuário já existente, recebe imediatamente o painel diário via `GET /internal/users/context-by-telegram`.

## Architectural Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Gerenciamento de Estado de Diálogo | `SessionStore` em memória no daemon do bot (`apps/bot`) protegido por `sync.RWMutex` | Mantém o bot desacoplado do banco de dados, sem roundtrips intermediários na API e sem persistir estados transientes incompletos no PostgreSQL. |
| Estratégia de Evicção de Sessões | Ticker periódico (`time.Ticker`) em background a cada 5 minutos expurgando sessões inativas há mais de 30 minutos | Evita vazamento de memória com onboarding abandonado e garante isolamento seguro por `telegram_id`. |
| Cancelamento e Reset | Comando `/cancelar` a qualquer momento durante o onboarding | Permite ao usuário abortar e limpar a sessão em memória imediatamente sem travar o estado da conversa. |
| Modelo de Interação no Chat | Híbrido: botões inline (`InlineKeyboardMarkup`) para escolhas estruturadas e texto livre com validação numérica estrita | Reduz digitação e fricção nas decisões (6x vs 12x, conclusão de despesas, pular/adicionar ativos) mantendo precisão nos valores. |
| Precisão Numérica e Validação | Parser tolerante a formatos pt-BR (`3500.00`, `3500,00`, `3500`, `R$ 3.500,00`) e conversão precisa para `decimal.Decimal` / `float64` | Impede erros de arredondamento e garante conformidade com regras contábeis bancárias (HalfEven). |
| Detecção de Usuário Cadastrado | Verificação idempotente no `/start` via `client.GetUserContextByTelegram` | Usuários que já configuraram o perfil recebem imediatamente o S2S do dia e menu de comandos, sem risco de sobrescrever dados acidentalmente. |
| Fronteira de Responsabilidade | `apps/bot` como cliente HTTP stateless da `apps/api` via `X-Internal-Secret` | Preserva a arquitetura monorepo: o bot apenas orquestra o diálogo e delega a matemática financeira e persistência para a API. |

## Stack Touched in Phase 1

- [x] Project scaffold (Go monorepo: `apps/bot`, `apps/api`, `make test-bot`, `make build-all`)
- [x] Routing — Telegram router em `apps/bot/internal/bot/bot.go` tratando `/start`, `/cancelar`, mensagens de texto e `CallbackQuery` de botões inline
- [x] Database / Backend API — `POST /internal/users/onboarding` (gravação de usuário, conta, categorias, despesas projetadas, metas e investimentos) e `GET /internal/users/context-by-telegram`
- [x] UI / Telegram Client — Diálogo conversacional formatado em Markdown com botões inline dinâmicos e badges visuais (`🟢 SAUDÁVEL`, `🟡 RESTRITO`)
- [x] Deployment / Local Run — Execução local padronizada via `make run-bot` e `make run-api`

## Out of Scope (Deferred to Later Slices)

- Comandos operacionais do motor preditivo: `/s2s`, `/gasto`, `/simular` what-if e `/checkin` diário (Fase 2 / Milestone 3).
- Gestão continuada e pós-onboarding da carteira de ações: `/investimento`, `/investimentos`, `/venda` (Fase 3 / Milestone 4).
- Workers agendados de notificações matinais (08:00) e alertas automáticos de degradação de ciclo (Fase 4 / Milestone 5).
- Interface web rica em Vue 3 com visualização gráfica da curva de liquidez futura e painel de investimentos (Fase 5 / Milestone 6).

## Subsequent Slice Plan

Cada fase subsequente adiciona uma fatia vertical sobre este esqueleto conversacional sem alterar as decisões arquiteturais:

- **Phase 2:** Comandos do Motor Preditivo & Operação Diária (`/s2s`, `/gasto`, `/simular`, `/checkin`)
- **Phase 3:** Gestão de Carteira de Ações no Telegram (`/investimento`, `/investimentos`, `/venda`)
- **Phase 4:** Notificações Proativas & Alertas de Risco (Workers agendados matinais e noturnos)
- **Phase 5:** Dashboard Web Completo & Gráficos 12 Ciclos (Frontend Vue 3 analítico)
