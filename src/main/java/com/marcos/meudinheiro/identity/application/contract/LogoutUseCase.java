package com.marcos.meudinheiro.identity.application.contract;

public interface LogoutUseCase {
  void execute(String rawRefreshToken);
}
