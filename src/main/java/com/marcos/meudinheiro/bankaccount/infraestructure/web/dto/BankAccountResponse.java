package com.marcos.meudinheiro.bankaccount.infraestructure.web.dto;

import com.marcos.meudinheiro.bankaccount.domain.enums.BankAccountType;

import java.math.BigDecimal;
import java.util.UUID;

public record BankAccountResponse(
        UUID id,
        String name,
        BankAccountType type,
        BigDecimal initialBalance,
        BigDecimal currentBalance
) {
}
