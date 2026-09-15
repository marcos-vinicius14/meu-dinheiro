package com.marcos.meudinheiro.bankaccount.application.contract.dto;

import com.marcos.meudinheiro.bankaccount.domain.enums.BankAccountType;
import java.math.BigDecimal;

public record CreateBankAccountInput(
    String name, BankAccountType type, BigDecimal initialBalance) {}
