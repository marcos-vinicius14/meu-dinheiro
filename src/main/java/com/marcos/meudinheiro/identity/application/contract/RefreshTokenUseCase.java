package com.marcos.meudinheiro.identity.application.contract;

import com.marcos.meudinheiro.identity.application.contract.dto.AuthenticationOutput;
import com.marcos.meudinheiro.shared.notification.OperationResult;

public interface RefreshTokenUseCase {
  OperationResult<AuthenticationOutput> execute(String refreshToken);
}
