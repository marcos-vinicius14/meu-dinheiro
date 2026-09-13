package com.marcos.meudinheiro.transaction.application.usecase;

import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.shared.valueobjects.Money;
import com.marcos.meudinheiro.transaction.application.contract.SimulatePurchaseUseCase;
import com.marcos.meudinheiro.transaction.application.contract.constant.TransactionMessages;
import com.marcos.meudinheiro.transaction.application.contract.dto.SimulationInput;
import com.marcos.meudinheiro.transaction.application.contract.dto.SimulationOutput;
import com.marcos.meudinheiro.transaction.domain.service.CurrentState;
import com.marcos.meudinheiro.transaction.domain.service.CycleContext;
import com.marcos.meudinheiro.transaction.domain.service.CycleSimulation;
import com.marcos.meudinheiro.transaction.domain.service.SimulatedPurchase;
import com.marcos.meudinheiro.transaction.domain.service.WhatIfSimulator;
import com.marcos.meudinheiro.transaction.domain.valueobject.DateInterval;
import org.springframework.stereotype.Service;

import java.time.YearMonth;
import java.util.List;
import java.util.UUID;

@Service
public class SimulatePurchaseUseCaseImpl implements SimulatePurchaseUseCase {

    private final WhatIfSimulator simulator;

    public SimulatePurchaseUseCaseImpl(WhatIfSimulator simulator) {
        this.simulator = simulator;
    }

    @Override
    public OperationResult<SimulationOutput> execute(UUID userId, SimulationInput input) {
        if (input.date() == null) {
            return OperationResult.failure("Data da simulação é obrigatória");
        }
        if (input.installments() == null || input.installments() <= 0) {
            return OperationResult.failure(TransactionMessages.BUNDLE_INSTALLMENTS_REQUIRED);
        }
        if (input.firstDueDate() == null) {
            return OperationResult.failure(TransactionMessages.BUNDLE_FIRST_DUE_DATE_REQUIRED);
        }

        var cycleInterval = new DateInterval(
                YearMonth.from(input.date()).atDay(1),
                YearMonth.from(input.date()).atEndOfMonth()
        );

        var context = new CycleContext(
                cycleInterval,
                input.date(),
                input.targetSavings() != null ? new Money(input.targetSavings()) : Money.zero(),
                input.flexibleBudgetCap() != null ? new Money(input.flexibleBudgetCap()) : Money.zero()
        );

        var state = new CurrentState(
                input.liquidBalance() != null ? new Money(input.liquidBalance()) : Money.zero(),
                List.of()
        );

        try {
            var purchase = new SimulatedPurchase(
                    input.totalAmount() != null ? new Money(input.totalAmount()) : Money.zero(),
                    input.installments(),
                    input.firstDueDate()
            );

            var simulations = simulator.simulate(context, state, purchase);

            var cycles = simulations.stream()
                    .map(SimulatePurchaseUseCaseImpl::toDto)
                    .toList();

            return OperationResult.success(new SimulationOutput(cycles));
        } catch (IllegalArgumentException e) {
            return OperationResult.failure(e.getMessage());
        }
    }

    private static SimulationOutput.CycleSimulationDto toDto(CycleSimulation simulation) {
        var result = simulation.engineResult();
        return new SimulationOutput.CycleSimulationDto(
                simulation.cycleInterval().startDate(),
                simulation.cycleInterval().endDate(),
                result.s2sToday().getValue(),
                simulation.s2sReduction().getValue(),
                simulation.s2sReductionPercent(),
                result.projectedBalance().getValue(),
                simulation.healthStatus().name(),
                simulation.bottleneck()
        );
    }
}
