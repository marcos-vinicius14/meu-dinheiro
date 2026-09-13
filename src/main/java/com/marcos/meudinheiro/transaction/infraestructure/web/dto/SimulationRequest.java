package com.marcos.meudinheiro.transaction.infraestructure.web.dto;

import java.math.BigDecimal;
import java.time.LocalDate;

public record SimulationRequest(
        BigDecimal liquidBalance,
        BigDecimal targetSavings,
        BigDecimal flexibleBudgetCap,
        LocalDate date,
        BigDecimal totalAmount,
        Integer installments,
        LocalDate firstDueDate
) {
}
