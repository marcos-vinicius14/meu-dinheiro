package com.marcos.meudinheiro.identity.infraestructure.security.authentication;

import com.github.benmanes.caffeine.cache.Cache;
import com.github.benmanes.caffeine.cache.Caffeine;
import com.marcos.meudinheiro.identity.application.contract.LoginAttemptLimiter;
import java.time.Duration;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

@Component
public class CaffeineLoginAttemptLimiter implements LoginAttemptLimiter {

  private final int maxAttempts;

  private final Cache<String, Integer> failedAttempts;

  public CaffeineLoginAttemptLimiter(
      @Value("${security.login.max-attempts:5}") int maxAttempts,
      @Value("${security.login.lockout:15m}") Duration lockout) {
    this.maxAttempts = maxAttempts;
    this.failedAttempts = Caffeine.newBuilder().expireAfterWrite(lockout).build();
  }

  @Override
  public boolean isBlocked(String email) {
    var attempts = failedAttempts.getIfPresent(email);

    return attempts != null && attempts >= maxAttempts;
  }

  @Override
  public void recordFailure(String email) {
    failedAttempts.asMap().compute(email, (key, attempts) -> attempts == null ? 1 : attempts + 1);
  }

  @Override
  public void reset(String email) {
    failedAttempts.invalidate(email);
  }
}
