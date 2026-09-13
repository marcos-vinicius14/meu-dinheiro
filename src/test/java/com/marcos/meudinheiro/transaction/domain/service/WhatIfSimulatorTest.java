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

class WhatIfSimulatorTest {

    private static final LocalDate CYCLE_START = LocalDate.of(2026, 9, 1);
    private static final LocalDate CYCLE_END = LocalDate.of(2026, 9, 30);
    private static final LocalDate TODAY = LocalDate.of(2026, 9, 13);

    private final WhatIfSimulator simulator = new WhatIfSimulator(new PredictiveEngine());

    private CycleContext baseContext() {
        return new CycleContext(
                new DateInterval(CYCLE_START, CYCLE_END),
                TODAY,
                money(100),
                money(900)
        );
    }

    private CurrentState baseState() {
        return new CurrentState(
                money(1000),
                List.of(
                        new TransactionSnapshot(
                                TransactionType.INCOME,
                                TransactionStatus.PROJECTED,
                                money(500),
                                LocalDate.of(2026, 9, 20)
                        ),
                        new TransactionSnapshot(
                                TransactionType.FIXED_EXPENSE,
                                TransactionStatus.PROJECTED,
                                money(300),
                                LocalDate.of(2026, 9, 15)
                        ),
                        new TransactionSnapshot(
                                TransactionType.INSTALLMENT_EXPENSE,
                                TransactionStatus.COMMITTED,
                                money(200),
                                LocalDate.of(2026, 9, 25)
                        )
                )
        );
    }

    @Test
    void singlePurchaseInCurrentCycleDegradesS2S() {
        // Base: NetLiquidity = 1000 + 500 - 300 - 200 - 100 = 900; R_flex = min(900, 900) = 900
        // S2S base = floor(900/18) = 50
        // Compra 300 à vista: NetLiquidity = 600; R_flex = min(900, 600) = 600
        // S2S = floor(600/18) = 33.33
        var result = simulator.simulate(
                baseContext(),
                baseState(),
                new SimulatedPurchase(money(300), 1, TODAY)
        );

        assertThat(result).hasSize(1);

        var first = result.get(0);
        assertThat(first.cycleInterval().startDate()).isEqualTo(CYCLE_START);
        assertThat(first.engineResult().s2sToday().getValue()).isEqualByComparingTo(new BigDecimal("33.33"));
        assertThat(first.s2sReduction().getValue()).isEqualByComparingTo(new BigDecimal("16.67"));
        assertThat(first.s2sReductionPercent()).isEqualByComparingTo(new BigDecimal("33.34"));
        // B_projected = 600 - 900 = -300 → RESTRICTED mesmo com S2S positivo
        assertThat(first.healthStatus()).isEqualTo(HealthStatus.RESTRICTED);
        assertThat(first.bottleneck()).isTrue();
    }

    @Test
    void installmentPurchaseProjectsAcrossSixCycles() {
        // Compra de 600 em 6x = 100/mês, começando em setembro
        // Setembro: NetLiquidity = 1000 + 500 - 300 - (200 + 100) - 100 = 800 → S2S = floor(800/18) = 44.44
        // Out/Nov/Dez/Jan/Fev: sem a renda/despesas de setembro (estado base só cobre setembro);
        // cada ciclo: liquidez 1000 - 100 (parcela) - 100 (reserva) = 800, cap 900 → S2S = floor(800/30)
        var result = simulator.simulate(
                baseContext(),
                baseState(),
                new SimulatedPurchase(money(600), 6, TODAY)
        );

        assertThat(result).hasSize(6);
        assertThat(result.get(0).cycleInterval().startDate()).isEqualTo(CYCLE_START);
        assertThat(result.get(1).cycleInterval().startDate()).isEqualTo(LocalDate.of(2026, 10, 1));
        assertThat(result.get(5).cycleInterval().startDate()).isEqualTo(LocalDate.of(2027, 2, 1));

        // Primeiro ciclo: D_rem = 18 → floor(800/18) = 44.44
        assertThat(result.get(0).engineResult().s2sToday().getValue())
                .isEqualByComparingTo(new BigDecimal("44.44"));
        // Ciclos futuros: liquidez 800; D_rem varia com o mês (out 31, nov 30, dez 31, jan 31, fev 28)
        // floor(800/31) = 25.80 | floor(800/30) = 26.66 | floor(800/28) = 28.57
        var expectedByCycle = new String[]{"25.80", "26.66", "25.80", "25.80", "28.57"};
        for (int i = 1; i < 6; i++) {
            assertThat(result.get(i).engineResult().s2sToday().getValue())
                    .isEqualByComparingTo(new BigDecimal(expectedByCycle[i - 1]));
        }
    }

    @Test
    void twelveMonthSimulationReportsBottleneckWhenLiquidityRunsOut() {
        // Liquidez 1000, compra de 2400 em 12x = 200/mês + reserva 100 + sem renda futura
        // Nos ciclos futuros: NetLiquidity = 1000 - 200 - 100 = 700 ≥ 0 → S2S = floor(700/30) = 23.33
        // Compra maior: 3600 em 12x = 300/mês → NetLiquidity = 1000 - 300 - 100 = 600 → S2S = 20
        // Compra de 24000 em 12x = 2000/mês → NetLiquidity = 1000 - 2000 - 100 < 0 → DEFICIT_RISK em todos
        var result = simulator.simulate(
                baseContext(),
                baseState(),
                new SimulatedPurchase(money(24000), 12, TODAY)
        );

        assertThat(result).hasSize(12);
        for (var cycle : result) {
            assertThat(cycle.healthStatus()).isEqualTo(HealthStatus.DEFICIT_RISK);
            assertThat(cycle.bottleneck()).isTrue();
        }
    }

    @Test
    void bottleneckIdentifiesCyclesThatDegradeHealth() {
        // Compra que deixa o ciclo atual saudável, mas estoura os futuros:
        // 12x de 700 = 58.34/mês; futuros: 1000 - 58.34 - 100 = 841.66 → saudável
        // Para gargalo: liquidez futura menor que o gasto comprometido
        // 12x de 1200 = 100/mês... liquidez 1000 → 800 → saudável também
        // Cenário: liquidez base futura é 1000 (sem renda projetada nos ciclos futuros);
        // compra 12x de 1300 = 108.34 → 1000 - 108.34 - 100 = 791.66 → saudável
        // Para gerar RESTRICTED nos futuros: B_projected < 0 → cap - parcela > liquidez...
        // cap 900: B = (1000 - parcela - 100) - 900 < 0 → parcela > 0 sempre que liquidez-reserva < cap
        // → todos os ciclos futuros ficam RESTRICTED (gargalo), o atual depende
        var result = simulator.simulate(
                baseContext(),
                baseState(),
                new SimulatedPurchase(money(12000), 12, TODAY)
        );

        assertThat(result).hasSize(12);
        // Ciclo atual: NetLiquidity = 900 - 1000 = -100 → DEFICIT (compra acima da liquidez!)
        assertThat(result.get(0).healthStatus()).isEqualTo(HealthStatus.DEFICIT_RISK);
        assertThat(result.get(0).bottleneck()).isTrue();
        // Ciclos futuros: 1000 - 1000 - 100 = -100 → DEFICIT
        for (int i = 1; i < 12; i++) {
            assertThat(result.get(i).healthStatus()).isEqualTo(HealthStatus.DEFICIT_RISK);
            assertThat(result.get(i).bottleneck()).isTrue();
        }
    }

    @Test
    void noImpactWhenPurchaseIsZero() {
        var result = simulator.simulate(
                baseContext(),
                baseState(),
                new SimulatedPurchase(money(0), 1, TODAY)
        );

        assertThat(result).hasSize(1);
        assertThat(result.get(0).s2sReduction().getValue()).isEqualByComparingTo(BigDecimal.ZERO);
    }

    private static Money money(int value) {
        return new Money(new BigDecimal(value));
    }
}
