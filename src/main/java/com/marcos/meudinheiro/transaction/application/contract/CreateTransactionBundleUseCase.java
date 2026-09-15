package com.marcos.meudinheiro.transaction.application.contract;

import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.transaction.application.contract.dto.CreateTransactionBundleInput;
import com.marcos.meudinheiro.transaction.application.contract.dto.TransactionBundleOutput;
import java.util.UUID;

public interface CreateTransactionBundleUseCase {
  OperationResult<TransactionBundleOutput> execute(UUID userId, CreateTransactionBundleInput input);
}
