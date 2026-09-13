# Motor de Predição (PredictiveEngine)

Estado atual da V1 preditiva: cálculo de **Safe-to-Spend (S2S)** diário, saúde
financeira do ciclo e simulação what-if de compras parceladas.

## Princípios

1. **Previsibilidade futura ativa**: despesas projetadas e parcelas comprometidas
   impactam a liquidez futura imediatamente — não é registro contábil retroativo.
2. **Precisão monetária**: `BigDecimal` com escala 2 e arredondamento `HALF_EVEN`;
   divisão diária do S2S por floor em centavos (`Money.toCents` → `floorDiv`).
   Nunca ponto flutuante.
3. **Domínio puro**: `PredictiveEngine` e `WhatIfSimulator` vivem em
   `transaction/domain/service`, sem Spring, sem I/O, determinísticos. Recebem
   `TransactionSnapshot` (record imutável), nunca a entity JPA. O wiring como bean
   fica em `infraestructure/configuration/PredictiveEngineConfig`.

## Componentes

| Tipo | Papel |
|---|---|
| `DateInterval` | Intervalo do ciclo; `daysRemaining(from)` = `max(1, end − from + 1)` |
| `CycleContext` | Intervalo do ciclo, data atual, `targetSavings`, `flexibleBudgetCap` |
| `CurrentState` | `liquidBalance` (caixa hoje) + lista de `TransactionSnapshot` |
| `TransactionSnapshot` | Visão imutável: `type`, `status`, `amount`, `dueDate` |
| `EngineResult` | `s2sToday`, `remainingFlexible`, `projectedBalance`, `netLiquidityBeforeFlex`, `flexibleSpent`, `projectedIncome`, `committedExpenses`, `healthStatus`, `daysRemaining` |
| `CycleSimulation` | Resultado por ciclo do simulador + `s2sReduction`, `s2sReductionPercent`, `bottleneck` |

## Tipos e status

- `TransactionType`: `INCOME`, `FIXED_EXPENSE`, `FLEXIBLE_EXPENSE`, `INSTALLMENT_EXPENSE`
- `TransactionStatus`: `PROJECTED` (previsão), `COMMITTED` (compromisso irrevogável,
  ex. parcela), `CONFIRMED` (liquidado), `CANCELED` (sem impacto)
- `HealthStatus`: `HEALTHY`, `RESTRICTED`, `DEFICIT_RISK`

## Algoritmo (`PredictiveEngine.calculate`)

Com `D_rem` = dias restantes do ciclo (mínimo 1, inclusive o dia atual):

1. **Agregações** (todas dentro do intervalo do ciclo):
   - `E_flex_spent`: despesas `FLEXIBLE_EXPENSE` `CONFIRMED` com vencimento
     **antes** de hoje (o gasto confirmado hoje entra via `spentToday` do check-in)
   - `I_proj`: receitas `INCOME` `[PROJECTED, COMMITTED]` de hoje em diante
   - `E_fixed`: despesas `FIXED_EXPENSE` `[PROJECTED, COMMITTED]` de hoje em diante
   - `E_committed`: parcelas `INSTALLMENT_EXPENSE` `COMMITTED` de hoje em diante
   - `CANCELED` é sempre ignorado; transações fora do ciclo também.

2. **Capacidade flexível restante**:
   ```
   RemainingBudget        = max(0, flexibleBudgetCap − E_flex_spent)
   NetLiquidityBeforeFlex = liquidBalance + I_proj − E_fixed − E_committed − targetSavings
   R_flex                 = min(RemainingBudget, NetLiquidityBeforeFlex)
   ```

3. **Balanço projetado**: `B_projected = NetLiquidityBeforeFlex − RemainingBudget`

4. **S2S e saúde**:
   - `R_flex ≤ 0` → `S2S = 0`, `DEFICIT_RISK`
   - `R_flex > 0` → `S2S = floor(R_flex / D_rem)` (por centavos),
     `HEALTHY` se `B_projected ≥ 0`, senão `RESTRICTED`

Note a distinção: `RESTRICTED` = dá para gastar algo hoje, mas o fim do ciclo fecha
negativo; `DEFICIT_RISK` = liquidez projetada não cobre nem os compromissos.

## Simulador What-If (`WhatIfSimulator.simulate`)

Entrada: `SimulatedPurchase(totalAmount, installments, firstDueDate)`.

1. Calcula o **baseline** (S2S sem a compra) antes de qualquer projeção.
2. Aloca o total em N parcelas com `Money.allocate` (HALF_EVEN; a **primeira**
   parcela absorve o resto de arredondamento — soma sempre exata).
3. Para cada ciclo mensal afetado: injeta a parcela como
   `INSTALLMENT_EXPENSE COMMITTED` no vencimento (mesmo day-of-month, herdado por
   `plusMonths`), recalcula com a engine e compara com o baseline.
4. Retorna por ciclo: `s2sToday`, `s2sReduction` (absoluta), `s2sReductionPercent`,
   `projectedBalance`, `healthStatus` e `bottleneck` (`RESTRICTED`/`DEFICIT_RISK`).

Sem persistência: é função pura sobre o estado informado.

## Check-in diário (`DailyCheckInUseCase`)

`POST /transactions/check-in`:

1. **Pré-valida tudo** (categorias, valores, ids pendentes) antes de qualquer
   escrita — um failure no meio da lista não persiste nada (evita escritas
   parciais dentro do `@Transactional` e duplicação em retry).
2. Persiste `untrackedExpenses` como `FLEXIBLE_EXPENSE CONFIRMED` na data.
3. Confirma as transações pendentes selecionadas (`confirm(date)`).
4. Executa a engine para o ciclo atual (mês calendário da data).
5. Upsert do snapshot em `tb_check_in_snapshots` (idempotente por
   `(user_id, check_in_date)`): `s2sCalculated`, `spentToday`,
   `deltaFromSafeToSpend`, `healthStatus`.
6. Retorna `nextDayS2S` (engine reexecutada para D+1) e `projectedFreeBalance`.

## Endpoints

| Rota | Função |
|---|---|
| `POST /transactions/check-in` | Encerramento do dia (acima) |
| `POST /transactions/simulations` | What-if sem persistência |
| `POST /transactions/bundles` | Cria parcelamento (N parcelas COMMITTED) |
| CRUD `/transactions` | Transações avulsas (parcela individual é imutável) |

## Limitações conhecidas (próximos passos)

- `liquidBalance` vem no request do check-in/simulação; a derivação automática
  a partir dos saldos das contas (`initialBalance` + transações CONFIRMED) ainda
  não está implementada.
- `targetSavings` e `flexibleBudgetCap` também vêm no request; configuração
  persistida por usuário é pendência.
- Ciclo = mês calendário; ciclo personalizado por usuário não suportado.
- Não há endpoint para cancelar um bundle inteiro (parcelas individuais são
  imutáveis por design).
- Despesas legadas migradas de `EXPENSE → FLEXIBLE_EXPENSE` (V5) contam para o
  teto flexível — se houver histórico real, reclassificar antes de migrar.
- Simulação considera apenas transações informadas no estado (a API atual simula
  sem carregar o histórico real do usuário).

## Testes

- `PredictiveEngineTest` (10): sem gasto flexível, estouro (recalibração p/ baixo),
  sobra (p/ cima), déficit por compromisso (`S2S = 0`), borda do último dia
  (`D_rem = 1`), receita confirmada não entra em `I_proj`, `CANCELED` ignorado,
  gasto fora do ciclo, reserva antes do teto.
- `WhatIfSimulatorTest` (5): compra à vista, 6x atravessando meses (D_rem varia:
  31/30/28), 12x com déficit em todos os ciclos, gargalos, compra zero.
- `MoneyTest` (11), `DateIntervalTest` (6), `TransactionModelTest` (14),
  `TransactionBundleModelTest` (8).
- ITs: `TransactionIT` (13), `DailyCheckInIT` (6), `WhatIfSimulationIT` (4).
