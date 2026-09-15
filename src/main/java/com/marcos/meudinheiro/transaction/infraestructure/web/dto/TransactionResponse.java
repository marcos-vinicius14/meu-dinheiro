package com.marcos.meudinheiro.transaction.infraestructure.web.dto;

import com.marcos.meudinheiro.transaction.application.contract.dto.TransactionOutput;
import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.UUID;
import org.jspecify.annotations.Nullable;

public record TransactionResponse(
    UUID id,
    String description,
    BigDecimal amount,
    String type,
    String status,
    @Nullable UUID categoryId,
    @Nullable UUID bankAccountId,
    LocalDate dueDate,
    @Nullable LocalDate paymentDate,
    @Nullable UUID bundleId,
    @Nullable Integer installmentNumber,
    @Nullable Integer totalInstallments) {

  public static TransactionResponse from(TransactionOutput output) {
    return new TransactionResponse(
        output.id(),
        output.description(),
        output.amount(),
        output.type().name(),
        output.status().name(),
        output.categoryId(),
        output.bankAccountId(),
        output.dueDate(),
        output.paymentDate(),
        output.bundleId(),
        output.installmentNumber(),
        output.totalInstallments());
  }
}
