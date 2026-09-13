package com.marcos.meudinheiro.transaction.infraestructure.web.dto;

import com.marcos.meudinheiro.transaction.application.contract.dto.TransactionOutput;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.UUID;

public record TransactionResponse(
        UUID id,
        String description,
        BigDecimal amount,
        String type,
        String status,
        UUID categoryId,
        UUID bankAccountId,
        LocalDate dueDate,
        LocalDate paymentDate,
        UUID bundleId,
        Integer installmentNumber,
        Integer totalInstallments
) {

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
                output.totalInstallments()
        );
    }
}
