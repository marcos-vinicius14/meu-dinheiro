package com.marcos.meudinheiro.bankaccount.application.mapper;

import com.marcos.meudinheiro.bankaccount.application.contract.dto.BankAccountOutput;
import com.marcos.meudinheiro.bankaccount.domain.model.BankAccountModel;
import com.marcos.meudinheiro.bankaccount.infraestructure.web.dto.BankAccountResponse;

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

    public static BankAccountResponse toResponse(BankAccountOutput output) {
        return new BankAccountResponse(
                output.id(),
                output.name(),
                output.type(),
                output.initialBalance(),
                output.currentBalance()
        );
    }
}
