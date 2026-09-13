package com.marcos.meudinheiro.transaction.application.contract;

import com.marcos.meudinheiro.bankaccount.domain.model.BankAccountModel;
import com.marcos.meudinheiro.shared.notification.OperationResult;

import java.util.UUID;

/**
 * Porta de acesso à conta bancária: valida ownership sem acessar repository alheio.
 */
public interface TransactionBankAccountResolver {
    OperationResult<BankAccountModel> resolve(UUID userId, UUID bankAccountId);
}
