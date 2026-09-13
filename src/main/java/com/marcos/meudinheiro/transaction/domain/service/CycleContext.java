package com.marcos.meudinheiro.transaction.domain.service;

import com.marcos.meudinheiro.shared.valueobjects.Money;
import com.marcos.meudinheiro.transaction.domain.valueobject.DateInterval;

public record CycleContext(
        DateInterval cycleInterval,
        java.time.LocalDate currentDate,
        Money targetSavings,
        Money flexibleBudgetCap
) {

    public CycleContext {
        if (cycleInterval == null || currentDate == null) {
            throw new IllegalArgumentException("Intervalo do ciclo e data atual são obrigatórios");
        }
        targetSavings = targetSavings != null ? targetSavings : Money.zero();
        flexibleBudgetCap = flexibleBudgetCap != null ? flexibleBudgetCap : Money.zero();
    }
}
