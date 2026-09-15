package com.marcos.meudinheiro.bankaccount.infraestructure.web.dto;

import com.marcos.meudinheiro.bankaccount.application.contract.constant.BankAccountMessages;
import jakarta.validation.constraints.NotBlank;
import java.math.BigDecimal;

public record UpdateBankAccountRequest(
    @NotBlank(message = BankAccountMessages.ACCOUNT_NAME_REQUIRED) String name,
    BigDecimal initialBalance) {}
