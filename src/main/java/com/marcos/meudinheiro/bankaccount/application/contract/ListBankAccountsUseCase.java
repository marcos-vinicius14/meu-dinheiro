package com.marcos.meudinheiro.bankaccount.application.contract;

import com.marcos.meudinheiro.bankaccount.application.contract.dto.BankAccountOutput;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import java.util.List;
import java.util.UUID;

public interface ListBankAccountsUseCase {
  OperationResult<List<BankAccountOutput>> execute(UUID userId);
}
