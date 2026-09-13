package com.marcos.meudinheiro.transaction.application.contract.dto;

import java.math.BigDecimal;
import java.time.LocalDate;

public record SimulationInput(
        BigDecimal liquidBalance,
        BigDecimal targetSavings,
        BigDecimal flexibleBudgetCap,
        LocalDate date,
        BigDecimal totalAmount,
        Integer installments,
        LocalDate firstDueDate
) {
}
