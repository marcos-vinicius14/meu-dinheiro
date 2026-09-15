package com.marcos.meudinheiro.transaction.application.usecase;

import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.shared.valueobjects.Money;
import com.marcos.meudinheiro.transaction.application.contract.TransactionBankAccountResolver;
import com.marcos.meudinheiro.transaction.application.contract.TransactionCategoryResolver;
import com.marcos.meudinheiro.transaction.application.contract.UpdateTransactionUseCase;
import com.marcos.meudinheiro.transaction.application.contract.constant.TransactionMessages;
import com.marcos.meudinheiro.transaction.application.contract.dto.TransactionOutput;
import com.marcos.meudinheiro.transaction.application.contract.dto.UpdateTransactionInput;
import com.marcos.meudinheiro.transaction.application.mapper.TransactionMapper;
import com.marcos.meudinheiro.transaction.domain.model.TransactionModel;
import com.marcos.meudinheiro.transaction.infraestructure.repository.TransactionRepository;
import java.util.UUID;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
public class UpdateTransactionUseCaseImpl implements UpdateTransactionUseCase {

  private final TransactionRepository repository;
  private final TransactionCategoryResolver categoryResolver;
  private final TransactionBankAccountResolver bankAccountResolver;

  public UpdateTransactionUseCaseImpl(
      TransactionRepository repository,
      TransactionCategoryResolver categoryResolver,
      TransactionBankAccountResolver bankAccountResolver) {
    this.repository = repository;
    this.categoryResolver = categoryResolver;
    this.bankAccountResolver = bankAccountResolver;
  }

  @Transactional
  @Override
  public OperationResult<TransactionOutput> execute(
      UUID userId, UUID transactionId, UpdateTransactionInput input) {
    return repository
        .findById(transactionId)
        .filter(transaction -> transaction.belongsTo(userId))
        .map(transaction -> updateTransaction(transaction, userId, input))
        .orElseGet(() -> OperationResult.failure(TransactionMessages.TRANSACTION_NOT_FOUND));
  }

  private OperationResult<TransactionOutput> updateTransaction(
      TransactionModel transaction, UUID userId, UpdateTransactionInput input) {
    if (transaction.getBundleId() != null) {
      return OperationResult.failure(
          "Parcelas de um parcelamento não podem ser alteradas individualmente");
    }

    // Validação completa antes de qualquer mutação da entidade gerenciada:
    // exception após mutação seria commitada pelo flush no fim da transação.
    if (input.type() == null) {
      return OperationResult.failure(TransactionMessages.TRANSACTION_TYPE_REQUIRED);
    }
    if (input.type()
        == com.marcos.meudinheiro.transaction.domain.enums.TransactionType.INSTALLMENT_EXPENSE) {
      return OperationResult.failure(TransactionMessages.TRANSACTION_TYPE_NOT_INSTALLMENT);
    }
    if (input.amount() == null || input.amount().signum() <= 0) {
      return OperationResult.failure(TransactionMessages.TRANSACTION_AMOUNT_REQUIRED);
    }
    if (input.dueDate() == null) {
      return OperationResult.failure(TransactionMessages.TRANSACTION_DUE_DATE_REQUIRED);
    }

    var categoryResult = categoryResolver.resolve(userId, input.categoryId());
    if (categoryResult.isFailure()) {
      return OperationResult.failure(categoryResult.errors());
    }
    var category = categoryResult.value();

    com.marcos.meudinheiro.bankaccount.domain.model.BankAccountModel bankAccount = null;
    if (input.bankAccountId() != null) {
      var bankAccountResult = bankAccountResolver.resolve(userId, input.bankAccountId());
      if (bankAccountResult.isFailure()) {
        return OperationResult.failure(bankAccountResult.errors());
      }
      bankAccount = bankAccountResult.value();
    }

    transaction.updateDescription(input.description());
    transaction.updateType(input.type());
    transaction.updateAmount(new Money(input.amount()));
    transaction.updateDueDate(input.dueDate());
    transaction.assignBankAccount(bankAccount);
    transaction.setCategoryRef(category);

    repository.save(transaction);

    return OperationResult.success(TransactionMapper.toOutput(transaction));
  }
}
