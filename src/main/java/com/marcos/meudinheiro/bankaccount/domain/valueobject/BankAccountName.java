package com.marcos.meudinheiro.bankaccount.domain.valueobject;

import com.marcos.meudinheiro.bankaccount.application.contract.constant.BankAccountMessages;
import com.marcos.meudinheiro.shared.notification.ValidationResult;

public record BankAccountName(String value) {

    private static final int MIN_LENGTH = 3;
    private static final int MAX_LENGTH = 100;

    public static ValidationResult<BankAccountName> create(String value) {
        if (value == null || value.isBlank()) {
            return ValidationResult.invalid(BankAccountMessages.ACCOUNT_NAME_REQUIRED);
        }

        var trimmed = value.trim();
        if (trimmed.length() < MIN_LENGTH || trimmed.length() > MAX_LENGTH) {
            return ValidationResult.invalid(BankAccountMessages.ACCOUNT_NAME_SIZE);
        }

        return ValidationResult.valid(new BankAccountName(trimmed));
    }
}
