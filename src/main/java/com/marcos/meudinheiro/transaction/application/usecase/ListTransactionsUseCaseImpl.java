package com.marcos.meudinheiro.transaction.application.usecase;

import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.transaction.application.contract.ListTransactionsUseCase;
import com.marcos.meudinheiro.transaction.application.contract.dto.TransactionOutput;
import com.marcos.meudinheiro.transaction.application.mapper.TransactionMapper;
import com.marcos.meudinheiro.transaction.infraestructure.repository.TransactionRepository;
import java.util.List;
import java.util.UUID;
import org.springframework.stereotype.Service;

@Service
public class ListTransactionsUseCaseImpl implements ListTransactionsUseCase {

  private final TransactionRepository repository;

  public ListTransactionsUseCaseImpl(TransactionRepository repository) {
    this.repository = repository;
  }

  @Override
  public OperationResult<List<TransactionOutput>> execute(UUID userId) {
    var transactions = repository.findAllByUserIdOrderByDueDateDesc(userId);

    var outputs = transactions.stream().map(TransactionMapper::toOutput).toList();

    return OperationResult.success(outputs);
  }
}
