package com.marcos.meudinheiro.bankaccount.infraestructure.configuration;

import com.marcos.meudinheiro.bankaccount.domain.model.BankAccountModel;
import com.marcos.meudinheiro.bankaccount.infraestructure.repository.BankAccountRepository;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.transaction.application.contract.TransactionBankAccountResolver;
import com.marcos.meudinheiro.transaction.application.contract.constant.TransactionMessages;
import java.util.UUID;
import org.springframework.stereotype.Component;

@Component
public class TransactionBankAccountResolverAdapter implements TransactionBankAccountResolver {

  private final BankAccountRepository bankAccountRepository;

  public TransactionBankAccountResolverAdapter(BankAccountRepository bankAccountRepository) {
    this.bankAccountRepository = bankAccountRepository;
  }

  @Override
  public OperationResult<BankAccountModel> resolve(UUID userId, UUID bankAccountId) {
    return bankAccountRepository
        .findById(bankAccountId)
        .filter(account -> account.belongsTo(userId))
        .map(OperationResult::success)
        .orElseGet(
            () -> OperationResult.failure(TransactionMessages.TRANSACTION_BANK_ACCOUNT_NOT_FOUND));
  }
}
