package com.marcos.meudinheiro.transaction.infraestructure.web.dto;

import com.marcos.meudinheiro.transaction.application.contract.constant.TransactionMessages;
import jakarta.validation.constraints.NotNull;
import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.UUID;

public record CreateTransactionRequest(
    String description,
    @NotNull(message = TransactionMessages.TRANSACTION_AMOUNT_REQUIRED) BigDecimal amount,
    @NotNull(message = TransactionMessages.TRANSACTION_TYPE_REQUIRED) String type,
    @NotNull(message = TransactionMessages.TRANSACTION_DUE_DATE_REQUIRED) LocalDate dueDate,
    @NotNull(message = TransactionMessages.TRANSACTION_CATEGORY_REQUIRED) UUID categoryId,
    UUID bankAccountId) {}
