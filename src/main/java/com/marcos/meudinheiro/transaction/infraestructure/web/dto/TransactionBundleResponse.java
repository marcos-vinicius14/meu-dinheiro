package com.marcos.meudinheiro.transaction.infraestructure.web.dto;

import com.marcos.meudinheiro.transaction.application.contract.dto.TransactionBundleOutput;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.UUID;

public record TransactionBundleResponse(
        UUID id,
        String description,
        BigDecimal totalAmount,
        Integer totalInstallments,
        LocalDate firstDueDate
) {

    public static TransactionBundleResponse from(TransactionBundleOutput output) {
        return new TransactionBundleResponse(
                output.id(),
                output.description(),
                output.totalAmount(),
                output.totalInstallments(),
                output.firstDueDate()
        );
    }
}
