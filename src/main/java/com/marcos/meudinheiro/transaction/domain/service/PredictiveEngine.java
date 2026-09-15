package com.marcos.meudinheiro.transaction.domain.service;

import com.marcos.meudinheiro.shared.valueobjects.Money;
import com.marcos.meudinheiro.transaction.domain.enums.HealthStatus;
import com.marcos.meudinheiro.transaction.domain.enums.TransactionStatus;
import com.marcos.meudinheiro.transaction.domain.enums.TransactionType;
import com.marcos.meudinheiro.transaction.domain.valueobject.DateInterval;
import java.time.LocalDate;
import java.util.List;

/**
 * Serviço de domínio puro: sem Spring, sem I/O, determinístico. Implementa as regras algorítmicas
 * de S2S (safe-to-spend) do ciclo.
 */
public class PredictiveEngine {

  public EngineResult calculate(CycleContext context, CurrentState state) {
    var interval = context.cycleInterval();
    var today = context.currentDate();

    var daysRemaining = interval.daysRemaining(today);

    var flexSpent = sumUntilYesterday(state.transactions(), today, interval);
    var projectedIncome = sumForward(state.transactions(), today, interval, TransactionType.INCOME);
    var fixedExpenses =
        sumForward(state.transactions(), today, interval, TransactionType.FIXED_EXPENSE);
    var committedExpenses = sumForwardCommittedInstallments(state.transactions(), today, interval);

    var remainingBudget = Money.max(context.flexibleBudgetCap().subtract(flexSpent), Money.zero());

    var netLiquidityBeforeFlex =
        state
            .liquidBalance()
            .add(projectedIncome)
            .subtract(fixedExpenses)
            .subtract(committedExpenses)
            .subtract(context.targetSavings());

    var remainingFlexible = Money.min(remainingBudget, netLiquidityBeforeFlex);

    var projectedBalance = netLiquidityBeforeFlex.subtract(remainingBudget);

    Money s2sToday;
    HealthStatus healthStatus;

    if (remainingFlexible.isNegative() || remainingFlexible.isZero()) {
      s2sToday = Money.zero();
      healthStatus = HealthStatus.DEFICIT_RISK;
    } else {
      s2sToday = floorPerDay(remainingFlexible, daysRemaining);
      healthStatus = projectedBalance.isNegative() ? HealthStatus.RESTRICTED : HealthStatus.HEALTHY;
    }

    return new EngineResult(
        s2sToday,
        Money.max(remainingFlexible, Money.zero()),
        projectedBalance,
        netLiquidityBeforeFlex,
        flexSpent,
        projectedIncome,
        fixedExpenses.add(committedExpenses),
        healthStatus,
        daysRemaining);
  }

  /** Despesas flexíveis CONFIRMADAS entre o início do ciclo e o dia anterior ao atual. */
  private Money sumUntilYesterday(
      List<TransactionSnapshot> transactions, LocalDate today, DateInterval interval) {
    return Money.sum(
        transactions.stream()
            .filter(t -> t.type() == TransactionType.FLEXIBLE_EXPENSE)
            .filter(t -> t.status() == TransactionStatus.CONFIRMED)
            .filter(t -> !t.dueDate().isBefore(interval.startDate()))
            .filter(t -> t.dueDate().isBefore(today))
            .map(TransactionSnapshot::amount)
            .toList());
  }

  /** Transações PROJECTED/COMMITTED do tipo informado entre hoje e o fim do ciclo. */
  private Money sumForward(
      List<TransactionSnapshot> transactions,
      LocalDate today,
      DateInterval interval,
      TransactionType type) {
    return Money.sum(
        transactions.stream()
            .filter(t -> t.type() == type)
            .filter(
                t ->
                    t.status() == TransactionStatus.PROJECTED
                        || t.status() == TransactionStatus.COMMITTED)
            .filter(t -> !t.dueDate().isBefore(today))
            .filter(t -> !t.dueDate().isAfter(interval.endDate()))
            .map(TransactionSnapshot::amount)
            .toList());
  }

  /** Parcelas COMMITTED de compras parceladas entre hoje e o fim do ciclo. */
  private Money sumForwardCommittedInstallments(
      List<TransactionSnapshot> transactions, LocalDate today, DateInterval interval) {
    return Money.sum(
        transactions.stream()
            .filter(t -> t.type() == TransactionType.INSTALLMENT_EXPENSE)
            .filter(t -> t.status() == TransactionStatus.COMMITTED)
            .filter(t -> !t.dueDate().isBefore(today))
            .filter(t -> !t.dueDate().isAfter(interval.endDate()))
            .map(TransactionSnapshot::amount)
            .toList());
  }

  /** Divisão diária com arredondamento para baixo em centavos (conservador). */
  private Money floorPerDay(Money total, int days) {
    var cents = total.toCents();
    var dailyCents = Math.floorDiv(cents, days);
    return Money.fromCents(dailyCents);
  }
}
