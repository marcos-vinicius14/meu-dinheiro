package com.marcos.meudinheiro.transaction.domain.service;

import com.marcos.meudinheiro.shared.valueobjects.Money;

import java.util.List;

public record CurrentState(
        Money liquidBalance,
        List<TransactionSnapshot> transactions
) {

    public CurrentState {
        liquidBalance = liquidBalance != null ? liquidBalance : Money.zero();
        transactions = transactions != null ? List.copyOf(transactions) : List.of();
    }
}
