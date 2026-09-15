package com.marcos.meudinheiro.transaction.application.usecase;

import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.transaction.application.contract.FindTransactionUseCase;
import com.marcos.meudinheiro.transaction.application.contract.constant.TransactionMessages;
import com.marcos.meudinheiro.transaction.application.contract.dto.TransactionOutput;
import com.marcos.meudinheiro.transaction.application.mapper.TransactionMapper;
import com.marcos.meudinheiro.transaction.infraestructure.repository.TransactionRepository;
import java.util.UUID;
import org.springframework.stereotype.Service;

@Service
public class FindTransactionUseCaseImpl implements FindTransactionUseCase {

  private final TransactionRepository repository;

  public FindTransactionUseCaseImpl(TransactionRepository repository) {
    this.repository = repository;
  }

  @Override
  public OperationResult<TransactionOutput> execute(UUID userId, UUID transactionId) {
    return repository
        .findById(transactionId)
        .filter(transaction -> transaction.belongsTo(userId))
        .map(TransactionMapper::toOutput)
        .map(OperationResult::success)
        .orElseGet(() -> OperationResult.failure(TransactionMessages.TRANSACTION_NOT_FOUND));
  }
}
