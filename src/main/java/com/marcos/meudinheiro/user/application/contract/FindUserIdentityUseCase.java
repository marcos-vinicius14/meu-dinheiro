package com.marcos.meudinheiro.user.application.contract;

import com.marcos.meudinheiro.user.application.contract.dto.UserIdentityOutput;

import java.util.Optional;

public interface FindUserIdentityUseCase {
    Optional<UserIdentityOutput> execute(String email);
}
