package com.marcos.meudinheiro.bankaccount.application.mapper;

import com.marcos.meudinheiro.bankaccount.application.contract.dto.BankAccountOutput;
import com.marcos.meudinheiro.bankaccount.domain.model.BankAccountModel;

public final class BankAccountMapper {

    private BankAccountMapper() {}

    public static BankAccountOutput toOutput(BankAccountModel account) {
        var initialBalance = account.getInitialBalance().getValue();
        return new BankAccountOutput(
                account.getId(),
                account.getName(),
                account.getBankAccountType(),
                initialBalance,
                initialBalance
        );
    }
}
