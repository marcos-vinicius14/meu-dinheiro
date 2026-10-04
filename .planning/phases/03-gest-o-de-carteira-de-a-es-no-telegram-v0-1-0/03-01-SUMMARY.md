# Summary 03-01: Endpoints Internos de Listagem Consolidada e Venda de Investimentos com Suporte no APIClient

## Resumo da Execução

A Wave 1 da Fase 3 foi concluída com sucesso em conformidade com as especificações de TDD, Troféu de Testes (PostgreSQL 18 real via Testcontainers) e segurança ASVS L1.

### 1. Testes de Integração Backend (RED -> GREEN)
- Criado `apps/api/tests/integration/internal_bot_api_investment_test.go` com 7 cenários completos:
  1. **Segurança e Acesso Restrito**: validação de recusa `403 Forbidden` sem `X-Internal-Secret` nos novos endpoints.
  2. **Consulta de Carteira Vazia (`GET /internal/investments`)**: retorno de lista vazia com saldo bancário líquido de R$ 5.000,00 e patrimônio de R$ 5.000,00.
  3. **Aporte de Múltiplos Ativos e Consulta Povoada**: adição de lotes de ALUP11 (recálculo de Preço Médio Ponderado para R$ 45,00), PETR4 e TD-SELIC (Renda Fixa), totalizando R$ 25.000,00 investidos e R$ 30.000,00 de patrimônio líquido total.
  4. **Venda Parcial (`POST /internal/investments/sell`)**: baixa de 50 cotas de ALUP11 preservando PM em R$ 45,00 e recalculando métricas globais.
  5. **Venda de 100% e Encerramento de Posição (D-09, D-12)**: encerramento total das cotas remanescentes de ALUP11, exclusão do banco de dados e remoção da lista de ativos ativos.
  6. **Recompra Inicia Novo Ciclo Limpo (D-11)**: nova compra de ALUP11 a R$ 60,00 iniciando ciclo limpo com PM estritamente em R$ 60,00.
  7. **Validações de Erro e Borda (T-03-02, T-03-03)**: tratamento de ticker inexistente (404), excesso de custódia (400), quantidade inválida (400) e usuário inexistente (404).

### 2. Implementação da API Go
- Em `apps/api/internal/botapi/investment.go`:
  - DTOs `ListInvestmentsResponse`, `SellInvestmentRequest`, `SellInvestmentResponse`.
  - Handler `handleListInvestments`: busca usuário por Telegram ID, obtém portfólio via `investService.List`, calcula liquidez via `userRepo.CalculateTotalLiquidBalance` e retorna fotografia consolidada.
  - Handler `handleSellInvestment`: validações defensivas, baixa atômica via `investService.Sell` (com locking `SELECT ... FOR UPDATE`), recálculo e identificação de `IsClosed`.
- Em `apps/api/internal/botapi/handler.go`:
  - Registro das rotas `GET /internal/investments` e `POST /internal/investments/sell` protegidas pelo middleware `requireInternalSecret`.

### 3. Implementação no APIClient do Bot
- Em `apps/bot/internal/client/types.go`:
  - Definição de `InvestmentItem`, `ListInvestmentsResponse`, `SellInvestmentRequest` e `SellInvestmentResponse` com suporte a `FlexFloat`.
- Em `apps/bot/internal/client/api_client.go`:
  - Implementação de `ListInvestments(ctx, telegramID)` e `SellInvestment(ctx, req)`.
- Em `apps/bot/internal/client/api_client_test.go`:
  - 4 novos testes unitários cobrindo requisições HTTP, headers, parsing de respostas e propagação de erros de validação e excesso de custódia.

## Verificação Automatizada
- `apps/api/tests/integration`: 100% aprovado contra Postgres 18 real (3.4s).
- `apps/bot/internal/client`: 100% aprovado com `-race` (1.0s).
- `make test-api`: 100% aprovado.
