package com.marcos.meudinheiro.transaction.application.contract;

import com.marcos.meudinheiro.transaction.application.contract.dto.TransactionOutput;
import com.marcos.meudinheiro.transaction.application.contract.dto.UpdateTransactionInput;
import com.marcos.meudinheiro.shared.notification.OperationResult;

import java.util.UUID;

public interface UpdateTransactionUseCase {
    OperationResult<TransactionOutput> execute(UUID userId, UUID transactionId, UpdateTransactionInput input);
}
