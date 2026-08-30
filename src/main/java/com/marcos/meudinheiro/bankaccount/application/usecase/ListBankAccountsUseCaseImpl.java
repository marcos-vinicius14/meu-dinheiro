package com.marcos.meudinheiro.bankaccount.application.usecase;

import com.marcos.meudinheiro.bankaccount.application.contract.ListBankAccountsUseCase;
import com.marcos.meudinheiro.bankaccount.application.contract.dto.BankAccountOutput;
import com.marcos.meudinheiro.bankaccount.application.mapper.BankAccountMapper;
import com.marcos.meudinheiro.bankaccount.infraestructure.repository.BankAccountRepository;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import org.springframework.stereotype.Service;

import java.util.List;
import java.util.UUID;

@Service
public class ListBankAccountsUseCaseImpl implements ListBankAccountsUseCase {

    private final BankAccountRepository repository;

    public ListBankAccountsUseCaseImpl(BankAccountRepository repository) {
        this.repository = repository;
    }

    @Override
    public OperationResult<List<BankAccountOutput>> execute(UUID userId) {
        var accounts = repository.findAllByUserId(userId);

        var outputs = accounts.stream()
                .map(BankAccountMapper::toOutput)
                .toList();

        return OperationResult.success(outputs);
    }
}
