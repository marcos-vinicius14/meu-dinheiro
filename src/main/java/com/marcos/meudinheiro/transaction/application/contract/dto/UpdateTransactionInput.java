package com.marcos.meudinheiro.transaction.application.contract.dto;

import com.marcos.meudinheiro.transaction.domain.enums.TransactionType;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.UUID;

public record UpdateTransactionInput(
        String description,
        BigDecimal amount,
        TransactionType type,
        LocalDate dueDate,
        UUID categoryId,
        UUID bankAccountId
) {
}
