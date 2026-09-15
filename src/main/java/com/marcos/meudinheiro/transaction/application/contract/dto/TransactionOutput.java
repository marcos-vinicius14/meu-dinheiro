package com.marcos.meudinheiro.transaction.application.contract.dto;

import com.marcos.meudinheiro.transaction.domain.enums.TransactionStatus;
import com.marcos.meudinheiro.transaction.domain.enums.TransactionType;
import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.UUID;
import org.jspecify.annotations.Nullable;

public record TransactionOutput(
    UUID id,
    String description,
    BigDecimal amount,
    TransactionType type,
    TransactionStatus status,
    @Nullable UUID categoryId,
    @Nullable UUID bankAccountId,
    LocalDate dueDate,
    @Nullable LocalDate paymentDate,
    @Nullable UUID bundleId,
    @Nullable Integer installmentNumber,
    @Nullable Integer totalInstallments) {}
