package com.marcos.meudinheiro.transaction.application.contract;

import com.marcos.meudinheiro.user.domain.model.UserModel;

import java.util.UUID;

public interface TransactionUserResolver {
    UserModel resolve(UUID userId);
}
