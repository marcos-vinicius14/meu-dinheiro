package com.marcos.meudinheiro.transaction.application.contract.dto;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.List;
import java.util.UUID;

public record DailyCheckInInput(
    LocalDate date,
    BigDecimal liquidBalance,
    BigDecimal targetSavings,
    BigDecimal flexibleBudgetCap,
    List<UntrackedExpense> untrackedExpenses,
    List<UUID> confirmedPendingTransactionIds) {

  public record UntrackedExpense(String description, BigDecimal amount, UUID categoryId) {}
}
