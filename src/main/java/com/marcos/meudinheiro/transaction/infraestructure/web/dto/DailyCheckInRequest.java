package com.marcos.meudinheiro.transaction.infraestructure.web.dto;

import jakarta.validation.constraints.NotNull;
import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.List;
import java.util.UUID;

public record DailyCheckInRequest(
    @NotNull LocalDate date,
    BigDecimal liquidBalance,
    BigDecimal targetSavings,
    BigDecimal flexibleBudgetCap,
    List<UntrackedExpenseRequest> untrackedExpenses,
    List<UUID> confirmedPendingTransactionIds) {

  public record UntrackedExpenseRequest(String description, BigDecimal amount, UUID categoryId) {}
}
