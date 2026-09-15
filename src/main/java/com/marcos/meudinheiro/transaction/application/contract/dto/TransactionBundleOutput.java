package com.marcos.meudinheiro.transaction.application.contract.dto;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.UUID;

public record TransactionBundleOutput(
    UUID id,
    String description,
    BigDecimal totalAmount,
    Integer totalInstallments,
    LocalDate firstDueDate) {}
