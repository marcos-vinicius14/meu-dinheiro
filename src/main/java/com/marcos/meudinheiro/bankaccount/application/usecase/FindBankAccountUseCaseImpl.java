package com.marcos.meudinheiro.bankaccount.application.usecase;

import com.marcos.meudinheiro.bankaccount.application.contract.FindBankAccountUseCase;
import com.marcos.meudinheiro.bankaccount.application.contract.constant.BankAccountMessages;
import com.marcos.meudinheiro.bankaccount.application.contract.dto.BankAccountOutput;
import com.marcos.meudinheiro.bankaccount.application.mapper.BankAccountMapper;
import com.marcos.meudinheiro.bankaccount.infraestructure.repository.BankAccountRepository;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import java.util.UUID;
import org.springframework.stereotype.Service;

@Service
public class FindBankAccountUseCaseImpl implements FindBankAccountUseCase {

  private final BankAccountRepository repository;

  public FindBankAccountUseCaseImpl(BankAccountRepository repository) {
    this.repository = repository;
  }

  @Override
  public OperationResult<BankAccountOutput> execute(UUID userId, UUID accountId) {
    return repository
        .findById(accountId)
        .filter(account -> account.belongsTo(userId))
        .map(BankAccountMapper::toOutput)
        .map(OperationResult::success)
        .orElseGet(() -> OperationResult.failure(BankAccountMessages.ACCOUNT_NOT_FOUND));
  }
}
