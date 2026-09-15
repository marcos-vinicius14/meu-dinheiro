package com.marcos.meudinheiro.identity.application.contract;

public interface LoginAttemptLimiter {
  boolean isBlocked(String email);

  void recordFailure(String email);

  void reset(String email);
}
