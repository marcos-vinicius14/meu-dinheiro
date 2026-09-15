package com.marcos.meudinheiro.identity.infraestructure.security.ratelimit;

import com.github.benmanes.caffeine.cache.Cache;
import com.github.benmanes.caffeine.cache.Caffeine;
import com.marcos.meudinheiro.identity.application.contract.LoginRateLimiter;
import java.time.Clock;
import java.time.Duration;
import java.time.Instant;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

@Component
public class CaffeineLoginRateLimiter implements LoginRateLimiter {

  private record Window(Instant start, int count) {}

  private final int maxRequests;
  private final Duration window;
  private final Clock clock;
  private final Cache<String, Window> windows;

  public CaffeineLoginRateLimiter(
      @Value("${security.login.rate-limit.max-requests:10}") int maxRequests,
      @Value("${security.login.rate-limit.window:1m}") Duration window,
      Clock clock) {
    this.maxRequests = maxRequests;
    this.window = window;
    this.clock = clock;
    this.windows = Caffeine.newBuilder().expireAfterWrite(window.multipliedBy(2)).build();
  }

  @Override
  public boolean tryAcquire(String clientIp) {
    var now = clock.instant();

    var current =
        windows
            .asMap()
            .compute(
                clientIp,
                (key, existing) -> {
                  if (existing == null || !now.isBefore(existing.start().plus(window))) {
                    return new Window(now, 1);
                  }

                  return new Window(existing.start(), existing.count() + 1);
                });

    return current.count() <= maxRequests;
  }

  @Override
  public void reset() {
    windows.invalidateAll();
  }
}
