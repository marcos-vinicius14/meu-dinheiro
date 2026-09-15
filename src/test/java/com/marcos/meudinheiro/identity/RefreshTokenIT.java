package com.marcos.meudinheiro.identity;

import static org.assertj.core.api.Assertions.assertThat;

import com.marcos.meudinheiro.IntegrationTestSupport;
import com.marcos.meudinheiro.identity.application.contract.RefreshTokenUseCase;
import com.marcos.meudinheiro.identity.infraestructure.security.token.RefreshTokenHash;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.JdbcTemplate;

class RefreshTokenIT extends IntegrationTestSupport {

  @Autowired private RefreshTokenUseCase refreshTokenUseCase;

  @Autowired private JdbcTemplate jdbcTemplate;

  @Test
  void rotatesRefreshTokenAndRevokesTheOldOne() throws Exception {
    var email = "refresh-rotate@example.com";
    var password = "password123";

    createUser(email, password);
    var session = login(email, password);

    var result = refreshTokenUseCase.execute(session.refreshToken());

    assertThat(result.isSuccess()).isTrue();

    var output = result.value();

    assertThat(output.acessToken()).isNotBlank();
    assertThat(output.tokenType()).isEqualTo("Bearer");
    assertThat(output.refreshToken()).isNotEqualTo(session.refreshToken());

    assertThat(activeTokenCount(session.refreshToken())).isZero();
    assertThat(revokedTokenCount(session.refreshToken())).isEqualTo(1);
    assertThat(activeTokenCount(output.refreshToken())).isEqualTo(1);
  }

  @Test
  void rotatedRefreshTokenCanBeRefreshedAgain() throws Exception {
    var email = "refresh-rotate-again@example.com";
    var password = "password123";

    createUser(email, password);
    var session = login(email, password);

    var firstRotation = refreshTokenUseCase.execute(session.refreshToken());

    var secondRotation = refreshTokenUseCase.execute(firstRotation.value().refreshToken());

    assertThat(secondRotation.value().refreshToken())
        .isNotEqualTo(firstRotation.value().refreshToken());
  }

  @Test
  void revokedRefreshTokenIsRejected() throws Exception {
    var email = "refresh-revoked@example.com";
    var password = "password123";

    createUser(email, password);
    var session = login(email, password);

    refreshTokenUseCase.execute(session.refreshToken());

    var result = refreshTokenUseCase.execute(session.refreshToken());

    assertThat(result.isFailure()).isTrue();
    assertThat(result.value()).isNull();
    assertThat(result.errors()).isNotEmpty();
  }

  @Test
  void unknownRefreshTokenIsRejected() {
    var result = refreshTokenUseCase.execute("not-a-real-refresh-token");

    assertThat(result.isFailure()).isTrue();
    assertThat(result.errors()).isNotEmpty();
  }

  @Test
  void expiredRefreshTokenIsRejected() throws Exception {
    var email = "refresh-expired@example.com";
    var password = "password123";

    createUser(email, password);
    var session = login(email, password);

    jdbcTemplate.update(
        "UPDATE tb_refresh_tokens SET expires_at = CURRENT_TIMESTAMP - INTERVAL '1 hour' WHERE token_hash = ?",
        RefreshTokenHash.sha256(session.refreshToken()));

    var result = refreshTokenUseCase.execute(session.refreshToken());

    assertThat(result.isFailure()).isTrue();
    assertThat(result.errors()).isNotEmpty();
  }

  @Test
  void refreshDoesNotTouchOtherSessions() throws Exception {
    var email = "refresh-other-sessions@example.com";
    var password = "password123";

    createUser(email, password);

    var firstSession = login(email, password);
    var secondSession = login(email, password);

    var result = refreshTokenUseCase.execute(firstSession.refreshToken());

    assertThat(result.isSuccess()).isTrue();

    assertThat(activeTokenCount(secondSession.refreshToken())).isEqualTo(1);
  }

  private int activeTokenCount(String rawRefreshToken) {
    return tokenCount(rawRefreshToken, "revoked_at IS NULL");
  }

  private int revokedTokenCount(String rawRefreshToken) {
    return tokenCount(rawRefreshToken, "revoked_at IS NOT NULL");
  }

  private int tokenCount(String rawRefreshToken, String revokedCondition) {
    return jdbcTemplate.queryForObject(
        "SELECT COUNT(*) FROM tb_refresh_tokens WHERE token_hash = ? AND " + revokedCondition,
        Integer.class,
        RefreshTokenHash.sha256(rawRefreshToken));
  }
}
