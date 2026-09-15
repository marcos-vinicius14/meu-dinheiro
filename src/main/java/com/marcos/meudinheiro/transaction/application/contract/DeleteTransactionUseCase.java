package com.marcos.meudinheiro.transaction.application.contract;

import com.marcos.meudinheiro.shared.notification.OperationResult;
import java.util.UUID;

public interface DeleteTransactionUseCase {
  OperationResult<Void> execute(UUID userId, UUID transactionId);
}
