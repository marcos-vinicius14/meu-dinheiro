package com.marcos.meudinheiro.bankaccount.application.usecase;

import com.marcos.meudinheiro.bankaccount.application.contract.UpdateBankAccountUseCase;
import com.marcos.meudinheiro.bankaccount.application.contract.constant.BankAccountMessages;
import com.marcos.meudinheiro.bankaccount.application.contract.dto.BankAccountOutput;
import com.marcos.meudinheiro.bankaccount.application.contract.dto.UpdateBankAccountInput;
import com.marcos.meudinheiro.bankaccount.application.mapper.BankAccountMapper;
import com.marcos.meudinheiro.bankaccount.domain.model.BankAccountModel;
import com.marcos.meudinheiro.bankaccount.domain.valueobject.BankAccountName;
import com.marcos.meudinheiro.bankaccount.infraestructure.repository.BankAccountRepository;
import com.marcos.meudinheiro.shared.notification.Notification;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.shared.valueobjects.Money;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.UUID;

@Service
public class UpdateBankAccountUseCaseImpl implements UpdateBankAccountUseCase {

    private final BankAccountRepository repository;

    public UpdateBankAccountUseCaseImpl(BankAccountRepository repository) {
        this.repository = repository;
    }

    @Transactional
    @Override
    public OperationResult<BankAccountOutput> execute(UUID userId, UUID accountId, UpdateBankAccountInput input) {
        return repository.findById(accountId)
                .filter(account -> account.belongsTo(userId))
                .map(account -> updateAccount(account, input))
                .orElseGet(() -> OperationResult.failure(BankAccountMessages.ACCOUNT_NOT_FOUND));
    }

    private OperationResult<BankAccountOutput> updateAccount(BankAccountModel account, UpdateBankAccountInput input) {
        var notification = new Notification();
        var name = notification.collect(BankAccountName.create(input.name()));

        if (notification.hasErrors()) {
            return OperationResult.failure(notification.errors());
        }

        account.updateName(name.value());

        if (input.initialBalance() != null) {
            account.updateInitialBalance(new Money(input.initialBalance()));
        }

        repository.save(account);

        return OperationResult.success(BankAccountMapper.toOutput(account));
    }
}
