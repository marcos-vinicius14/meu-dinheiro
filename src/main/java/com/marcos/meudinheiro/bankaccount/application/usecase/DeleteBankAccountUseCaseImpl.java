package com.marcos.meudinheiro.bankaccount.application.usecase;

import com.marcos.meudinheiro.bankaccount.application.contract.DeleteBankAccountUseCase;
import com.marcos.meudinheiro.bankaccount.application.contract.constant.BankAccountMessages;
import com.marcos.meudinheiro.bankaccount.domain.model.BankAccountModel;
import com.marcos.meudinheiro.bankaccount.infraestructure.repository.BankAccountRepository;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import java.util.UUID;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
public class DeleteBankAccountUseCaseImpl implements DeleteBankAccountUseCase {

  private final BankAccountRepository repository;

  public DeleteBankAccountUseCaseImpl(BankAccountRepository repository) {
    this.repository = repository;
  }

  @Transactional
  @Override
  public OperationResult<Void> execute(UUID userId, UUID accountId) {
    return repository
        .findById(accountId)
        .filter(account -> account.belongsTo(userId))
        .map(account -> deleteAccount(accountId, account))
        .orElseGet(() -> OperationResult.failure(BankAccountMessages.ACCOUNT_NOT_FOUND));
  }

  private OperationResult<Void> deleteAccount(UUID accountId, BankAccountModel account) {
    if (repository.hasTransactions(accountId)) {
      return OperationResult.failure(BankAccountMessages.ACCOUNT_HAS_TRANSACTIONS);
    }

    repository.delete(account);

    return OperationResult.success();
  }
}
