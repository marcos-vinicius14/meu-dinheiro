package com.marcos.meudinheiro.bankaccount.application.contract.dto;

import com.marcos.meudinheiro.bankaccount.domain.enums.BankAccountType;
import java.math.BigDecimal;
import java.util.UUID;

public record BankAccountOutput(
    UUID id,
    String name,
    BankAccountType type,
    BigDecimal initialBalance,
    BigDecimal currentBalance) {}
