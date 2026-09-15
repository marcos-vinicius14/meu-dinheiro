package com.marcos.meudinheiro.transaction.application.contract.dto;

import com.marcos.meudinheiro.transaction.domain.enums.TransactionType;
import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.UUID;
import org.jspecify.annotations.Nullable;

public record UpdateTransactionInput(
    String description,
    BigDecimal amount,
    @Nullable TransactionType type,
    LocalDate dueDate,
    UUID categoryId,
    UUID bankAccountId) {}
