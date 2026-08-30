package com.marcos.meudinheiro.bankaccount.application.contract.dto;

import java.math.BigDecimal;

public record UpdateBankAccountInput(
        String name,
        BigDecimal initialBalance
) {
}
