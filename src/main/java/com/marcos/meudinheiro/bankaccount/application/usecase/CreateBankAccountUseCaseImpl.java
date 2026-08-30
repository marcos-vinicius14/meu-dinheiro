package com.marcos.meudinheiro.bankaccount.application.usecase;

import com.marcos.meudinheiro.bankaccount.application.contract.BankAccountUserResolver;
import com.marcos.meudinheiro.bankaccount.application.contract.CreateBankAccountUseCase;
import com.marcos.meudinheiro.bankaccount.application.contract.constant.BankAccountMessages;
import com.marcos.meudinheiro.bankaccount.application.contract.dto.BankAccountOutput;
import com.marcos.meudinheiro.bankaccount.application.contract.dto.CreateBankAccountInput;
import com.marcos.meudinheiro.bankaccount.application.mapper.BankAccountMapper;
import com.marcos.meudinheiro.bankaccount.domain.model.BankAccountModel;
import com.marcos.meudinheiro.bankaccount.domain.valueobject.BankAccountName;
import com.marcos.meudinheiro.bankaccount.infraestructure.repository.BankAccountRepository;
import com.marcos.meudinheiro.shared.notification.Notification;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.shared.notification.ValidationResult;
import com.marcos.meudinheiro.shared.valueobjects.Money;

import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.math.BigDecimal;
import java.util.UUID;

@Service
public class CreateBankAccountUseCaseImpl implements CreateBankAccountUseCase {

    private static final int MAX_ACCOUNTS_PER_USER = 3;

    private final BankAccountRepository repository;
    private final BankAccountUserResolver userResolver;

    public CreateBankAccountUseCaseImpl(
            BankAccountRepository repository,
            BankAccountUserResolver userResolver
    ) {
        this.repository = repository;
        this.userResolver = userResolver;
    }

    @Transactional
    @Override
    public OperationResult<BankAccountOutput> execute(UUID userId, CreateBankAccountInput input) {
        repository.acquireCreationLock(creationLockId(userId));

        var validation = validateBankAccount(userId, input);

        if (validation.isInvalid()) {
            return OperationResult.failure(validation.errors());
        }
        
        var accountName = validation.value();

        var initialBalance = input.initialBalance() != null
                ? input.initialBalance()
                : BigDecimal.ZERO;

        var account = BankAccountModel.create(
                accountName.value(),
                input.type(),
                new Money(initialBalance),
                userResolver.resolve(userId)
        );

        repository.save(account);

        return OperationResult.success(BankAccountMapper.toOutput(account));
    }

   
    private ValidationResult<BankAccountName> validateBankAccount(
        UUID userId,
        CreateBankAccountInput input
    ) {
        var notification = new Notification();

        var name = notification.collect(
            BankAccountName.create(input.name())
        );

        if (input.type() == null) {
            notification.collect(ValidationResult.invalid(BankAccountMessages.ACCOUNT_TYPE_REQUIRED));
        }

        if (repository.countByUserId(userId) >= MAX_ACCOUNTS_PER_USER) {
            notification.collect(ValidationResult.invalid(BankAccountMessages.ACCOUNT_LIMIT_REACHED));
        }

        if (notification.hasErrors()) {
            return ValidationResult.invalid(notification.errors());
        }
    
        return ValidationResult.valid(name);

    }

    private static long creationLockId(UUID userId) {
        return userId.getMostSignificantBits() ^ userId.getLeastSignificantBits();
    }
}
