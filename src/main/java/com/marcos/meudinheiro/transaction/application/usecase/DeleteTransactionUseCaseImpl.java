package com.marcos.meudinheiro.transaction.application.usecase;

import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.transaction.application.contract.DeleteTransactionUseCase;
import com.marcos.meudinheiro.transaction.application.contract.constant.TransactionMessages;
import com.marcos.meudinheiro.transaction.infraestructure.repository.TransactionRepository;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.UUID;

@Service
public class DeleteTransactionUseCaseImpl implements DeleteTransactionUseCase {

    private final TransactionRepository repository;

    public DeleteTransactionUseCaseImpl(TransactionRepository repository) {
        this.repository = repository;
    }

    @Transactional
    @Override
    public OperationResult<Void> execute(UUID userId, UUID transactionId) {
        return repository.findById(transactionId)
                .filter(transaction -> transaction.belongsTo(userId))
                .map(transaction -> {
                    repository.delete(transaction);
                    return OperationResult.success();
                })
                .orElseGet(() -> OperationResult.failure(TransactionMessages.TRANSACTION_NOT_FOUND));
    }
}
