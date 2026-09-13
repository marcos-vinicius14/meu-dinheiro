package com.marcos.meudinheiro.transaction.infraestructure.configuration;

import com.marcos.meudinheiro.bankaccount.domain.model.BankAccountModel;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.transaction.application.contract.TransactionBankAccountResolver;
import com.marcos.meudinheiro.transaction.application.contract.constant.TransactionMessages;
import com.marcos.meudinheiro.bankaccount.infraestructure.repository.BankAccountRepository;
import org.springframework.stereotype.Component;

import java.util.UUID;

@Component
public class TransactionBankAccountResolverAdapter implements TransactionBankAccountResolver {

    private final BankAccountRepository bankAccountRepository;

    public TransactionBankAccountResolverAdapter(BankAccountRepository bankAccountRepository) {
        this.bankAccountRepository = bankAccountRepository;
    }

    @Override
    public OperationResult<BankAccountModel> resolve(UUID userId, UUID bankAccountId) {
        return bankAccountRepository.findById(bankAccountId)
                .filter(account -> account.belongsTo(userId))
                .map(OperationResult::success)
                .orElseGet(() -> OperationResult.failure(TransactionMessages.TRANSACTION_BANK_ACCOUNT_NOT_FOUND));
    }
}
