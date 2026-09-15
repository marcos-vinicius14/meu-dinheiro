package com.marcos.meudinheiro.identity.application.contract;

import com.marcos.meudinheiro.shared.notification.OperationResult;

public interface LogoutAllUseCase {
  OperationResult<Void> execute();
}
