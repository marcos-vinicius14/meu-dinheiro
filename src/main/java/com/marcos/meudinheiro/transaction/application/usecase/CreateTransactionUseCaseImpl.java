package com.marcos.meudinheiro.transaction.application.usecase;

import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.transaction.application.contract.CreateTransactionUseCase;
import com.marcos.meudinheiro.transaction.application.contract.TransactionBankAccountResolver;
import com.marcos.meudinheiro.transaction.application.contract.TransactionCategoryResolver;
import com.marcos.meudinheiro.transaction.application.contract.TransactionUserResolver;
import com.marcos.meudinheiro.transaction.application.contract.constant.TransactionMessages;
import com.marcos.meudinheiro.transaction.application.contract.dto.CreateTransactionInput;
import com.marcos.meudinheiro.transaction.application.contract.dto.TransactionOutput;
import com.marcos.meudinheiro.transaction.application.mapper.TransactionMapper;
import com.marcos.meudinheiro.transaction.domain.model.TransactionModel;
import com.marcos.meudinheiro.transaction.infraestructure.repository.TransactionRepository;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.UUID;

@Service
public class CreateTransactionUseCaseImpl implements CreateTransactionUseCase {

    private final TransactionRepository repository;
    private final TransactionUserResolver userResolver;
    private final TransactionCategoryResolver categoryResolver;
    private final TransactionBankAccountResolver bankAccountResolver;

    public CreateTransactionUseCaseImpl(
            TransactionRepository repository,
            TransactionUserResolver userResolver,
            TransactionCategoryResolver categoryResolver,
            TransactionBankAccountResolver bankAccountResolver
    ) {
        this.repository = repository;
        this.userResolver = userResolver;
        this.categoryResolver = categoryResolver;
        this.bankAccountResolver = bankAccountResolver;
    }

    @Transactional
    @Override
    public OperationResult<TransactionOutput> execute(UUID userId, CreateTransactionInput input) {
        if (input.type() == null) {
            return OperationResult.failure(TransactionMessages.TRANSACTION_TYPE_REQUIRED);
        }

        var categoryResult = categoryResolver.resolve(userId, input.categoryId());
        if (categoryResult.isFailure()) {
            return OperationResult.failure(categoryResult.errors());
        }
        var category = categoryResult.value();

        var categoryError = validateCategoryType(category, input.type());
        if (categoryError != null) {
            return OperationResult.failure(categoryError);
        }

        com.marcos.meudinheiro.bankaccount.domain.model.BankAccountModel bankAccount = null;
        if (input.bankAccountId() != null) {
            var bankAccountResult = bankAccountResolver.resolve(userId, input.bankAccountId());
            if (bankAccountResult.isFailure()) {
                return OperationResult.failure(bankAccountResult.errors());
            }
            bankAccount = bankAccountResult.value();
        }

        try {
            var transaction = TransactionModel.create(
                    input.description(),
                    input.type(),
                    input.amount() != null
                            ? new com.marcos.meudinheiro.shared.valueobjects.Money(input.amount())
                            : null,
                    input.dueDate(),
                    userResolver.resolve(userId),
                    bankAccount,
                    category
            );

            repository.save(transaction);

            return OperationResult.success(TransactionMapper.toOutput(transaction));
        } catch (IllegalArgumentException e) {
            return OperationResult.failure(e.getMessage());
        }
    }

    private String validateCategoryType(
            com.marcos.meudinheiro.category.domain.model.CategoryModel category,
            com.marcos.meudinheiro.transaction.domain.enums.TransactionType type
    ) {
        boolean flexibleType = type == com.marcos.meudinheiro.transaction.domain.enums.TransactionType.FLEXIBLE_EXPENSE;
        if (flexibleType && !category.isFlexible()) {
            return TransactionMessages.TRANSACTION_CATEGORY_INVALID_TYPE;
        }
        if (!flexibleType && category.isFlexible()) {
            return TransactionMessages.TRANSACTION_CATEGORY_INVALID_TYPE;
        }
        return null;
    }
}
