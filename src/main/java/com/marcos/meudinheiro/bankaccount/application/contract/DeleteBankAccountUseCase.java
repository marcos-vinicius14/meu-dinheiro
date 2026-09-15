package com.marcos.meudinheiro.bankaccount.application.contract;

import com.marcos.meudinheiro.shared.notification.OperationResult;
import java.util.UUID;

public interface DeleteBankAccountUseCase {
  OperationResult<Void> execute(UUID userId, UUID accountId);
}
