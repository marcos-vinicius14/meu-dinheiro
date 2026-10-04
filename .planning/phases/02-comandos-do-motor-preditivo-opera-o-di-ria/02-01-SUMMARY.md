# 02-01-SUMMARY: Endpoints Internos da API e Client Layer HTTP

## Visão Geral
Conclusão da primeira onda da Fase 2 (Comandos do Motor Preditivo & Operação Diária), disponibilizando os endpoints internos na API REST Go e seus respectivos métodos com testes unitários no `APIClient` do bot do Telegram.

---

## Entregas Realizadas

### 1. Testes de Integração com Postgres 18 via Testcontainers (TDD Estrito)
- Arquivo: `apps/api/tests/integration/internal_bot_api_predictive_test.go`
- Suíte completa cobrindo:
  - Proteção de segurança com token interno (`X-Internal-Secret`).
  - Simulação What-If parcelada (12x) e à vista com identificação determinística do ciclo crítico e recalibração de S2S.
  - Check-in diário com idempotência (`UPSERT` em `tb_check_in_snapshots`) e agregação cumulativa de despesas flexíveis do dia via `SumFlexibleExpensesByDate`.
  - Ajuste de saldo de conta corrente com recálculo automático de liquidez total no contexto financeiro do usuário.
  - Tratamento de erro 404 para usuários inexistentes e 400 para payloads inválidos.

### 2. Endpoints Internos no Handler da API (`apps/api/internal/botapi`)
- `POST /internal/transactions/simulations` (`simulation.go`): executa simulações what-if sem persistência ao longo de até 12 ciclos mensais, retornando variação de S2S, saúde do ciclo crítico e alerta de déficit.
- `POST /internal/transactions/checkin` (`checkin.go`): registra despesas não rastreadas, soma despesas do dia, executa o motor preditivo e persiste o snapshot diário de forma atômica.
- `POST /internal/bank-accounts/balance` (`balance.go`): atualiza o saldo bancário da conta especificada (ou da conta padrão) e retorna a nova liquidez consolidada.
- Registro das rotas em `handler.go` com injeção do `transaction.Service`.

### 3. Camada do Repositório e Serviço de Transações (`apps/api/internal/transaction`)
- Implementado `SumFlexibleExpensesByDate` em `repository.go`.
- Ajustado `DailyCheckIn` em `service.go` para agregar todas as despesas flexíveis confirmadas do dia.
- Carregamento de transações ativas do usuário em `SimulatePurchase` para projeção precisa de compromissos futuros.

### 4. Camada de Cliente HTTP no Bot (`apps/bot/internal/client`)
- Adicionados métodos e DTOs em `api_client.go`:
  - `SimulatePurchase(ctx, SimulationRequest) (*SimulationResponse, error)`
  - `DailyCheckIn(ctx, DailyCheckInRequest) (*DailyCheckInResponse, error)`
  - `AdjustAccountBalance(ctx, AdjustBalanceRequest) (*AdjustBalanceResponse, error)`
- Testes unitários com mocks HTTP em `api_client_test.go` cobrindo cenários de sucesso, validação de payload e erros HTTP (400, 403, 500).

---

## Verificação e Qualidade
- `go test -v -tags=integration -run TestInternalBotAPI_Predictive ./tests/integration/...` -> **PASS**
- `go test -v -race ./apps/api/internal/...` -> **PASS**
- `go test -v -race ./apps/bot/internal/client/...` -> **PASS**
- `make test-bot` -> **PASS**
