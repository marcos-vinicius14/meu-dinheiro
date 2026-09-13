package com.marcos.meudinheiro.transaction.domain.service;

import static org.assertj.core.api.Assertions.assertThat;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.List;

import com.marcos.meudinheiro.shared.valueobjects.Money;
import com.marcos.meudinheiro.transaction.domain.enums.HealthStatus;
import com.marcos.meudinheiro.transaction.domain.enums.TransactionStatus;
import com.marcos.meudinheiro.transaction.domain.enums.TransactionType;
import com.marcos.meudinheiro.transaction.domain.valueobject.DateInterval;
import org.junit.jupiter.api.Test;

class PredictiveEngineTest {

    private static final LocalDate CYCLE_START = LocalDate.of(2026, 9, 1);
    private static final LocalDate CYCLE_END = LocalDate.of(2026, 9, 30);
    private static final LocalDate TODAY = LocalDate.of(2026, 9, 13);

    private final PredictiveEngine engine = new PredictiveEngine();

    private CycleContext context(Money targetSavings, Money flexibleBudgetCap) {
        return new CycleContext(
                new DateInterval(CYCLE_START, CYCLE_END),
                TODAY,
                targetSavings,
                flexibleBudgetCap
        );
    }

    private TransactionSnapshot confirmedFlexible(Money amount, LocalDate date) {
        return snapshot(
                TransactionType.FLEXIBLE_EXPENSE,
                TransactionStatus.CONFIRMED,
                amount,
                date
        );
    }

    private TransactionSnapshot projected(TransactionType type, Money amount, LocalDate dueDate) {
        return snapshot(type, TransactionStatus.PROJECTED, amount, dueDate);
    }

    private TransactionSnapshot committed(TransactionType type, Money amount, LocalDate dueDate) {
        return snapshot(type, TransactionStatus.COMMITTED, amount, dueDate);
    }

    @Test
    void noFlexibleSpendingSoFarGivesFullDailyAllowance() {
        // D_rem = 18, cap = 900, gasto flex = 0 → RemainingBudget = 900
        // Liquidez: 1000 + 500 (renda projetada) - 300 (fixa) - 200 (parcela) - 100 (reserva) = 900
        // R_flex = min(900, 900) = 900 → S2S = floor(900/18) = 50
        var result = engine.calculate(
                context(money(100), money(900)),
                new CurrentState(
                        money(1000),
                        List.of(
                                projected(TransactionType.INCOME, money(500), LocalDate.of(2026, 9, 20)),
                                projected(TransactionType.FIXED_EXPENSE, money(300), LocalDate.of(2026, 9, 15)),
                                committed(TransactionType.INSTALLMENT_EXPENSE, money(200), LocalDate.of(2026, 9, 25))
                        )
                )
        );

        assertThat(result.s2sToday().getValue()).isEqualByComparingTo(new BigDecimal("50.00"));
        assertThat(result.remainingFlexible().getValue()).isEqualByComparingTo(new BigDecimal("900.00"));
        assertThat(result.projectedBalance().getValue()).isEqualByComparingTo(BigDecimal.ZERO);
        assertThat(result.healthStatus()).isEqualTo(HealthStatus.HEALTHY);
        assertThat(result.daysRemaining()).isEqualTo(18);
    }

    @Test
    void overspendYesterdayRecalibratesDown() {
        // Cap 900, já gastou 800 em flex → RemainingBudget = 100
        // NetLiquidity = 1000 + 0 - 0 - 0 - 0 = 1000 → R_flex = min(100, 1000) = 100
        // S2S = floor(100/18) = 5.55 → 5.55 (floor em centavos)
        var result = engine.calculate(
                context(money(0), money(900)),
                new CurrentState(
                        money(1000),
                        List.of(
                                confirmedFlexible(money(800), LocalDate.of(2026, 9, 12))
                        )
                )
        );

        assertThat(result.remainingFlexible().getValue()).isEqualByComparingTo(new BigDecimal("100.00"));
        assertThat(result.s2sToday().getValue()).isEqualByComparingTo(new BigDecimal("5.55"));
        assertThat(result.healthStatus()).isEqualTo(HealthStatus.HEALTHY);
    }

    @Test
    void underspendYesterdayRecalibratesUp() {
        // Cap 900, gastou só 100 → RemainingBudget = 800
        // NetLiquidity = 500 → R_flex = min(800, 500) = 500
        // S2S = floor(500/18) = 27.77
        var result = engine.calculate(
                context(money(0), money(900)),
                new CurrentState(
                        money(500),
                        List.of(
                                confirmedFlexible(money(100), LocalDate.of(2026, 9, 12))
                        )
                )
        );

        assertThat(result.remainingFlexible().getValue()).isEqualByComparingTo(new BigDecimal("500.00"));
        assertThat(result.s2sToday().getValue()).isEqualByComparingTo(new BigDecimal("27.77"));
    }

    @Test
    void projectedDeficitFromCommitmentsForcesZeroS2SAndDeficitRisk() {
        // NetLiquidity = 100 + 200 - 500 - 400 - 0 = -600 → R_flex = min(300, -600) < 0
        // → S2S = 0, DEFICIT_RISK
        var result = engine.calculate(
                context(money(0), money(300)),
                new CurrentState(
                        money(100),
                        List.of(
                                projected(TransactionType.INCOME, money(200), LocalDate.of(2026, 9, 20)),
                                projected(TransactionType.FIXED_EXPENSE, money(500), LocalDate.of(2026, 9, 15)),
                                committed(TransactionType.INSTALLMENT_EXPENSE, money(400), LocalDate.of(2026, 9, 25))
                        )
                )
        );

        assertThat(result.s2sToday().getValue()).isEqualByComparingTo(BigDecimal.ZERO);
        assertThat(result.healthStatus()).isEqualTo(HealthStatus.DEFICIT_RISK);
        assertThat(result.remainingFlexible().getValue()).isEqualByComparingTo(BigDecimal.ZERO);
    }

    @Test
    void negativeProjectedBalanceButPositiveFlexCapacityIsRestricted() {
        // NetLiquidity = 100 + 0 - 200 - 0 - 0 = -100 → R_flex = min(300, -100) < 0 → DEFICIT
        // Ajuste: cap 300, liquidez 400, fixed 200 → Net = 200 ≥ 0 → R_flex = 200
        // B_projected = 200 - 300 = -100 < 0 → RESTRICTED
        var result = engine.calculate(
                context(money(0), money(300)),
                new CurrentState(
                        money(400),
                        List.of(
                                projected(TransactionType.FIXED_EXPENSE, money(200), LocalDate.of(2026, 9, 15))
                        )
                )
        );

        assertThat(result.s2sToday().getValue()).isEqualByComparingTo(new BigDecimal("11.11"));
        assertThat(result.healthStatus()).isEqualTo(HealthStatus.RESTRICTED);
        assertThat(result.projectedBalance().getValue()).isEqualByComparingTo(new BigDecimal("-100.00"));
    }

    @Test
    void lastDayOfCycleHasSingleDayRemaining() {
        var lastDayContext = new CycleContext(
                new DateInterval(CYCLE_START, CYCLE_END),
                CYCLE_END,
                money(0),
                money(300)
        );

        var result = engine.calculate(
                lastDayContext,
                new CurrentState(
                        money(300),
                        List.of()
                )
        );

        assertThat(result.daysRemaining()).isEqualTo(1);
        assertThat(result.s2sToday().getValue()).isEqualByComparingTo(new BigDecimal("300.00"));
    }

    @Test
    void confirmedIncomeBeforeTodayCountsTowardLiquidBalanceNotProjectedIncome() {
        // Receita CONFIRMED de 500 no dia 5 NÃO entra em I_proj (que só considera PROJECTED/COMMITTED)
        // NetLiquidity = 1000 + 0 - 0 - 0 - 0 = 1000 → R_flex = min(900, 1000) = 900
        var result = engine.calculate(
                context(money(0), money(900)),
                new CurrentState(
                        money(1000),
                        List.of(
                                snapshot(
                                        TransactionType.INCOME,
                                        TransactionStatus.CONFIRMED,
                                        money(500),
                                        LocalDate.of(2026, 9, 5)
                                )
                        )
                )
        );

        assertThat(result.remainingFlexible().getValue()).isEqualByComparingTo(new BigDecimal("900.00"));
    }

    @Test
    void canceledTransactionsAreIgnored() {
        var result = engine.calculate(
                context(money(0), money(900)),
                new CurrentState(
                        money(1000),
                        List.of(
                                snapshot(
                                        TransactionType.FIXED_EXPENSE,
                                        TransactionStatus.CANCELED,
                                        money(10000),
                                        LocalDate.of(2026, 9, 15)
                                )
                        )
                )
        );

        assertThat(result.remainingFlexible().getValue()).isEqualByComparingTo(new BigDecimal("900.00"));
        assertThat(result.healthStatus()).isEqualTo(HealthStatus.HEALTHY);
    }

    @Test
    void flexibleSpendingOutsideCycleDoesNotConsumeBudget() {
        var result = engine.calculate(
                context(money(0), money(900)),
                new CurrentState(
                        money(1000),
                        List.of(
                                confirmedFlexible(money(500), LocalDate.of(2026, 8, 15))
                        )
                )
        );

        assertThat(result.remainingFlexible().getValue()).isEqualByComparingTo(new BigDecimal("900.00"));
    }

    @Test
    void targetSavingsIsReserveBeforeFlexibleBudget() {
        // NetLiquidity = 1000 - 400 (reserva) = 600; cap 900 → R_flex = 600
        // B_projected = 600 - 900 = -300 → RESTRICTED
        var result = engine.calculate(
                context(money(400), money(900)),
                new CurrentState(money(1000), List.of())
        );

        assertThat(result.remainingFlexible().getValue()).isEqualByComparingTo(new BigDecimal("600.00"));
        assertThat(result.healthStatus()).isEqualTo(HealthStatus.RESTRICTED);
    }

    private static TransactionSnapshot snapshot(
            TransactionType type,
            TransactionStatus status,
            Money amount,
            LocalDate dueDate
    ) {
        return new TransactionSnapshot(type, status, amount, dueDate);
    }

    private static Money money(int value) {
        return new Money(new BigDecimal(value));
    }
}
