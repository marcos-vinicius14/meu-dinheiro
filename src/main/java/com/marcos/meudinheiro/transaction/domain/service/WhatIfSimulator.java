package com.marcos.meudinheiro.transaction.domain.service;

import com.marcos.meudinheiro.shared.valueobjects.Money;
import com.marcos.meudinheiro.transaction.domain.enums.HealthStatus;
import com.marcos.meudinheiro.transaction.domain.enums.TransactionStatus;
import com.marcos.meudinheiro.transaction.domain.enums.TransactionType;
import com.marcos.meudinheiro.transaction.domain.valueobject.DateInterval;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.YearMonth;
import java.util.ArrayList;
import java.util.List;

/**
 * Serviço de domínio puro: simula o impacto de uma compra ao longo de N ciclos mensais
 * sem persistência. Reutiliza a PredictiveEngine para cada ciclo projetado.
 */
public class WhatIfSimulator {

    private final PredictiveEngine engine;

    public WhatIfSimulator(PredictiveEngine engine) {
        this.engine = engine;
    }

    public List<CycleSimulation> simulate(
            CycleContext baseContext,
            CurrentState baseState,
            SimulatedPurchase purchase
    ) {
        var allocations = purchase.totalAmount().allocate(purchase.installments());

        var firstCycleMonth = YearMonth.from(baseContext.cycleInterval().startDate());
        var purchaseStartMonth = YearMonth.from(purchase.firstDueDate());

        var simulations = new ArrayList<CycleSimulation>();

        var baselineResult = engine.calculate(baseContext, baseState);
        var baselineS2S = baselineResult.s2sToday();

        for (int i = 0; i < purchase.installments(); i++) {
            var cycleMonth = firstCycleMonth.plusMonths(i);
            var cycleInterval = new DateInterval(
                    cycleMonth.atDay(1),
                    cycleMonth.atEndOfMonth()
            );
            var cycleCurrentDate = cycleMonth.equals(firstCycleMonth)
                    ? baseContext.currentDate()
                    : cycleInterval.startDate();

            var transactions = new ArrayList<TransactionSnapshot>();

            for (TransactionSnapshot t : baseState.transactions()) {
                transactions.add(new TransactionSnapshot(t.type(), t.status(), t.amount(), t.dueDate()));
            }

            var installmentDueDate = installmentDueDateInCycle(
                    purchase.firstDueDate(),
                    purchaseStartMonth,
                    cycleMonth
            );
            if (installmentDueDate != null) {
                transactions.add(new TransactionSnapshot(
                        TransactionType.INSTALLMENT_EXPENSE,
                        TransactionStatus.COMMITTED,
                        allocations.get(i),
                        installmentDueDate
                ));
            }

            var effectiveState = cycleMonth.equals(firstCycleMonth)
                    ? new CurrentState(baseState.liquidBalance(), transactions)
                    : futureState(baseState, transactions);

            var context = new CycleContext(
                    cycleInterval,
                    cycleCurrentDate,
                    baseContext.targetSavings(),
                    baseContext.flexibleBudgetCap()
            );

            var result = engine.calculate(context, effectiveState);

            var reduction = baselineS2S.subtract(result.s2sToday());
            var reductionPercent = baselineS2S.isZero()
                    ? BigDecimal.ZERO
                    : reduction.percentOver(baselineS2S);

            var bottleneck = result.healthStatus() == HealthStatus.DEFICIT_RISK
                    || result.healthStatus() == HealthStatus.RESTRICTED;

            simulations.add(new CycleSimulation(
                    cycleInterval,
                    result,
                    Money.max(reduction, Money.zero()),
                    reductionPercent,
                    result.healthStatus(),
                    bottleneck
            ));
        }

        return simulations;
    }

    /**
     * Data da parcela deste ciclo: mesmo day-of-month da primeira parcela,
     * ajustado para o fim do mês quando o dia não existe.
     */
    private LocalDate installmentDueDateInCycle(
            LocalDate firstDueDate,
            YearMonth purchaseStartMonth,
            YearMonth cycleMonth
    ) {
        var monthsBetween = (int) java.time.temporal.ChronoUnit.MONTHS.between(purchaseStartMonth, cycleMonth);
        if (monthsBetween < 0) {
            return null;
        }
        return firstDueDate.plusMonths(monthsBetween);
    }

    /**
     * Estado dos ciclos futuros: sem liquidez herdada dos eventos do ciclo atual.
     * Usa a liquidez base como caixa inicial e apenas as transações recorrentes/parcelas.
     */
    private CurrentState futureState(CurrentState baseState, List<TransactionSnapshot> transactions) {
        return new CurrentState(baseState.liquidBalance(), transactions);
    }
}
