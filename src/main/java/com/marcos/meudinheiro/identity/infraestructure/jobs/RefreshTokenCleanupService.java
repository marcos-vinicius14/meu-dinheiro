package com.marcos.meudinheiro.identity.infraestructure.jobs;

import com.marcos.meudinheiro.identity.infraestructure.repository.RefreshTokenRepository;
import java.time.Clock;
import java.time.Duration;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
public class RefreshTokenCleanupService {

  private final RefreshTokenRepository refreshTokenRepository;
  private final Duration retention;
  private final Clock clock;

  public RefreshTokenCleanupService(
      RefreshTokenRepository refreshTokenRepository,
      @Value("${security.refresh-token.retention:7d}") Duration retention,
      Clock clock) {
    this.refreshTokenRepository = refreshTokenRepository;
    this.retention = retention;
    this.clock = clock;
  }

  @Scheduled(cron = "0 0 3 * * *")
  @Transactional
  public void cleanupExpiredTokens() {
    var cutoff = clock.instant().minus(retention);

    refreshTokenRepository.deleteStale(cutoff);
  }
}
