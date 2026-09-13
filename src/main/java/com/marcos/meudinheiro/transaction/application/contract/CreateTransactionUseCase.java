package com.marcos.meudinheiro.transaction.application.contract;

import com.marcos.meudinheiro.transaction.application.contract.dto.CreateTransactionInput;
import com.marcos.meudinheiro.transaction.application.contract.dto.TransactionOutput;
import com.marcos.meudinheiro.shared.notification.OperationResult;

import java.util.UUID;

public interface CreateTransactionUseCase {
    OperationResult<TransactionOutput> execute(UUID userId, CreateTransactionInput input);
}
