package com.marcos.meudinheiro.transaction.application.contract.dto;

import com.marcos.meudinheiro.transaction.domain.enums.TransactionStatus;
import com.marcos.meudinheiro.transaction.domain.enums.TransactionType;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.UUID;

public record TransactionOutput(
        UUID id,
        String description,
        BigDecimal amount,
        TransactionType type,
        TransactionStatus status,
        UUID categoryId,
        UUID bankAccountId,
        LocalDate dueDate,
        LocalDate paymentDate,
        UUID bundleId,
        Integer installmentNumber,
        Integer totalInstallments
) {
}
