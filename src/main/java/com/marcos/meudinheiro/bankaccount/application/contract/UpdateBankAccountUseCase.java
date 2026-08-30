package com.marcos.meudinheiro.bankaccount.application.contract;

import com.marcos.meudinheiro.bankaccount.application.contract.dto.BankAccountOutput;
import com.marcos.meudinheiro.bankaccount.application.contract.dto.UpdateBankAccountInput;
import com.marcos.meudinheiro.shared.notification.OperationResult;

import java.util.UUID;

public interface UpdateBankAccountUseCase {
    OperationResult<BankAccountOutput> execute(UUID userId, UUID accountId, UpdateBankAccountInput input);
}
