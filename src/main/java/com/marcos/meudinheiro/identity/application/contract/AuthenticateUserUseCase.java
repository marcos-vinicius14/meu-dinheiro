package com.marcos.meudinheiro.identity.application.contract;

import com.marcos.meudinheiro.identity.application.contract.dto.AuthenticationInput;
import com.marcos.meudinheiro.identity.application.contract.dto.AuthenticationOutput;

public interface AuthenticateUserUseCase {
    AuthenticationOutput execute(AuthenticationInput authenticationInput);
}
