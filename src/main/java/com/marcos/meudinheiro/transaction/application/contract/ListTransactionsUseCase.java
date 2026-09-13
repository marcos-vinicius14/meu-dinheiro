package com.marcos.meudinheiro.transaction.application.contract;

import com.marcos.meudinheiro.transaction.application.contract.dto.TransactionOutput;
import com.marcos.meudinheiro.shared.notification.OperationResult;

import java.util.List;
import java.util.UUID;

public interface ListTransactionsUseCase {
    OperationResult<List<TransactionOutput>> execute(UUID userId);
}
