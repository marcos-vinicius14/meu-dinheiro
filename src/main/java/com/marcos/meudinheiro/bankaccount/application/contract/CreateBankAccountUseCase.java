package com.marcos.meudinheiro.bankaccount.application.contract;

import com.marcos.meudinheiro.bankaccount.application.contract.dto.BankAccountOutput;
import com.marcos.meudinheiro.bankaccount.application.contract.dto.CreateBankAccountInput;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import java.util.UUID;

public interface CreateBankAccountUseCase {
  OperationResult<BankAccountOutput> execute(UUID userId, CreateBankAccountInput input);
}
