package com.marcos.meudinheiro.transaction.application.usecase;

import com.marcos.meudinheiro.bankaccount.domain.model.BankAccountModel;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.shared.valueobjects.Money;
import com.marcos.meudinheiro.transaction.application.contract.CreateTransactionBundleUseCase;
import com.marcos.meudinheiro.transaction.application.contract.TransactionBankAccountResolver;
import com.marcos.meudinheiro.transaction.application.contract.TransactionCategoryResolver;
import com.marcos.meudinheiro.transaction.application.contract.TransactionUserResolver;
import com.marcos.meudinheiro.transaction.application.contract.constant.TransactionMessages;
import com.marcos.meudinheiro.transaction.application.contract.dto.CreateTransactionBundleInput;
import com.marcos.meudinheiro.transaction.application.contract.dto.TransactionBundleOutput;
import com.marcos.meudinheiro.transaction.domain.model.TransactionBundleModel;
import com.marcos.meudinheiro.transaction.infraestructure.repository.TransactionBundleRepository;
import com.marcos.meudinheiro.transaction.infraestructure.repository.TransactionRepository;
import java.util.Objects;
import java.util.UUID;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
public class CreateTransactionBundleUseCaseImpl implements CreateTransactionBundleUseCase {

  private final TransactionBundleRepository bundleRepository;
  private final TransactionRepository transactionRepository;
  private final TransactionUserResolver userResolver;
  private final TransactionCategoryResolver categoryResolver;
  private final TransactionBankAccountResolver bankAccountResolver;

  public CreateTransactionBundleUseCaseImpl(
      TransactionBundleRepository bundleRepository,
      TransactionRepository transactionRepository,
      TransactionUserResolver userResolver,
      TransactionCategoryResolver categoryResolver,
      TransactionBankAccountResolver bankAccountResolver) {
    this.bundleRepository = bundleRepository;
    this.transactionRepository = transactionRepository;
    this.userResolver = userResolver;
    this.categoryResolver = categoryResolver;
    this.bankAccountResolver = bankAccountResolver;
  }

  @Transactional
  @Override
  public OperationResult<TransactionBundleOutput> execute(
      UUID userId, CreateTransactionBundleInput input) {
    if (input.totalInstallments() == null || input.totalInstallments() <= 0) {
      return OperationResult.failure(TransactionMessages.BUNDLE_INSTALLMENTS_REQUIRED);
    }

    var categoryResult = categoryResolver.resolve(userId, input.categoryId());
    if (categoryResult.isFailure()) {
      return OperationResult.failure(categoryResult.errors());
    }
    com.marcos.meudinheiro.category.domain.model.CategoryModel category = categoryResult.value();

    BankAccountModel bankAccount = null;
    if (input.bankAccountId() != null) {
      var bankAccountResult = bankAccountResolver.resolve(userId, input.bankAccountId());
      if (bankAccountResult.isFailure()) {
        return OperationResult.failure(bankAccountResult.errors());
      }
      bankAccount = bankAccountResult.value();
    }

    var user = userResolver.resolve(userId);

    try {
      var bundle =
          TransactionBundleModel.create(
              input.description(),
              input.totalAmount() != null ? new Money(input.totalAmount()) : null,
              input.totalInstallments(),
              input.firstDueDate(),
              user);

      var persisted = bundleRepository.save(bundle);

      var installments = persisted.generateInstallments(persisted.getId(), user, category);
      for (var installment : installments) {
        installment.assignBankAccount(bankAccount);
        transactionRepository.save(installment);
      }

      return OperationResult.success(
          new TransactionBundleOutput(
              persisted.getId(),
              persisted.getDescription(),
              persisted.getTotalAmount().getValue(),
              persisted.getTotalInstallments(),
              persisted.getFirstDueDate()));
    } catch (IllegalArgumentException e) {
      return OperationResult.failure(Objects.requireNonNull(e.getMessage()));
    }
  }
}
