package com.marcos.meudinheiro.bankaccount.infraestructure.web.dto;

import com.marcos.meudinheiro.bankaccount.application.contract.constant.BankAccountMessages;
import com.marcos.meudinheiro.bankaccount.domain.enums.BankAccountType;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;

import java.math.BigDecimal;

public record CreateBankAccountRequest(
        @NotBlank(message = BankAccountMessages.ACCOUNT_NAME_REQUIRED)
        String name,

        @NotNull(message = BankAccountMessages.ACCOUNT_TYPE_REQUIRED)
        BankAccountType type,

        BigDecimal initialBalance
) {
}
