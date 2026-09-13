package com.marcos.meudinheiro.transaction.domain.service;

import com.marcos.meudinheiro.shared.valueobjects.Money;
import com.marcos.meudinheiro.transaction.domain.enums.TransactionStatus;
import com.marcos.meudinheiro.transaction.domain.enums.TransactionType;

import java.time.LocalDate;

/**
 * Visão imutável de uma transação para os serviços de domínio puros.
 * Desacopla a engine da entidade JPA.
 */
public record TransactionSnapshot(
        TransactionType type,
        TransactionStatus status,
        Money amount,
        LocalDate dueDate
) {

    public TransactionSnapshot {
        if (type == null || status == null || amount == null || dueDate == null) {
            throw new IllegalArgumentException("Transação incompleta para cálculo");
        }
    }
}
