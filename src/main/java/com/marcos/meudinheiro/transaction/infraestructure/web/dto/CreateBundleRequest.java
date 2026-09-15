package com.marcos.meudinheiro.transaction.infraestructure.web.dto;

import com.marcos.meudinheiro.transaction.application.contract.constant.TransactionMessages;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.UUID;

public record CreateBundleRequest(
    @NotBlank(message = TransactionMessages.BUNDLE_NOT_FOUND) String description,
    @NotNull(message = TransactionMessages.BUNDLE_TOTAL_AMOUNT_REQUIRED) BigDecimal totalAmount,
    @NotNull(message = TransactionMessages.BUNDLE_INSTALLMENTS_REQUIRED) Integer totalInstallments,
    @NotNull(message = TransactionMessages.BUNDLE_FIRST_DUE_DATE_REQUIRED) LocalDate firstDueDate,
    @NotNull(message = TransactionMessages.TRANSACTION_CATEGORY_REQUIRED) UUID categoryId,
    UUID bankAccountId) {}
