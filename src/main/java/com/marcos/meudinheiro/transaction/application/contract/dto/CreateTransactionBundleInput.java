package com.marcos.meudinheiro.transaction.application.contract.dto;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.UUID;

public record CreateTransactionBundleInput(
    String description,
    BigDecimal totalAmount,
    Integer totalInstallments,
    LocalDate firstDueDate,
    UUID categoryId,
    UUID bankAccountId) {}
