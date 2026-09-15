package com.marcos.meudinheiro.identity.infraestructure.security.ratelimit;

import static org.assertj.core.api.Assertions.assertThat;

import java.time.Clock;
import java.time.Duration;
import java.time.Instant;
import java.time.ZoneId;
import java.time.ZoneOffset;
import org.junit.jupiter.api.Test;

class CaffeineLoginRateLimiterTest {

  private static final Instant START = Instant.parse("2026-01-01T00:00:00Z");

  @Test
  void allowsUpToMaxRequestsPerWindow() {
    var clock = new MutableClock();
    var limiter = new CaffeineLoginRateLimiter(2, Duration.ofMinutes(1), clock);

    assertThat(limiter.tryAcquire("10.0.0.1")).isTrue();
    assertThat(limiter.tryAcquire("10.0.0.1")).isTrue();
    assertThat(limiter.tryAcquire("10.0.0.1")).isFalse();
    assertThat(limiter.tryAcquire("10.0.0.1")).isFalse();
  }

  @Test
  void windowResetAllowsRequestsAgain() {
    var clock = new MutableClock();
    var limiter = new CaffeineLoginRateLimiter(2, Duration.ofMinutes(1), clock);

    assertThat(limiter.tryAcquire("10.0.0.1")).isTrue();
    assertThat(limiter.tryAcquire("10.0.0.1")).isTrue();
    assertThat(limiter.tryAcquire("10.0.0.1")).isFalse();

    clock.advance(Duration.ofMinutes(1).plusSeconds(1));

    assertThat(limiter.tryAcquire("10.0.0.1")).isTrue();
  }

  @Test
  void countersAreIndependentPerIp() {
    var clock = new MutableClock();
    var limiter = new CaffeineLoginRateLimiter(2, Duration.ofMinutes(1), clock);

    assertThat(limiter.tryAcquire("10.0.0.1")).isTrue();
    assertThat(limiter.tryAcquire("10.0.0.1")).isTrue();
    assertThat(limiter.tryAcquire("10.0.0.1")).isFalse();

    assertThat(limiter.tryAcquire("10.0.0.2")).isTrue();
  }

  @Test
  void requestsJustInsideWindowDoNotReset() {
    var clock = new MutableClock();
    var limiter = new CaffeineLoginRateLimiter(2, Duration.ofMinutes(1), clock);

    assertThat(limiter.tryAcquire("10.0.0.1")).isTrue();

    clock.advance(Duration.ofSeconds(30));

    assertThat(limiter.tryAcquire("10.0.0.1")).isTrue();
    assertThat(limiter.tryAcquire("10.0.0.1")).isFalse();
  }

  private static final class MutableClock extends Clock {

    private Instant now = START;

    void advance(Duration duration) {
      now = now.plus(duration);
    }

    @Override
    public Instant instant() {
      return now;
    }

    @Override
    public ZoneId getZone() {
      return ZoneOffset.UTC;
    }

    @Override
    public Clock withZone(ZoneId zone) {
      return this;
    }
  }
}
