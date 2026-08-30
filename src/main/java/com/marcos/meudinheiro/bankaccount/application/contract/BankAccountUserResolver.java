package com.marcos.meudinheiro.bankaccount.application.contract;

import com.marcos.meudinheiro.user.domain.model.UserModel;

import java.util.UUID;

public interface BankAccountUserResolver {
    UserModel resolve(UUID userId);
}
