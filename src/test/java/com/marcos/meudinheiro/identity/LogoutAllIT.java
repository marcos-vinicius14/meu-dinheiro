package com.marcos.meudinheiro.identity;

import static org.assertj.core.api.Assertions.assertThat;

import com.marcos.meudinheiro.IntegrationTestSupport;
import com.marcos.meudinheiro.identity.infraestructure.security.token.RefreshTokenHash;
import org.junit.jupiter.api.Test;

class LogoutAllIT extends IntegrationTestSupport {

  @Test
  void logoutAllRevokesEverySessionOfCurrentUser() {
    var email = "logout-all@example.com";
    var password = "password123";

    createUser(email, password);

    var firstSession = login(email, password);
    var secondSession = login(email, password);

    var response = authenticated(firstSession).post("/auth/logout-all", null);

    assertThat(response.status()).isEqualTo(204);

    assertThat(activeTokenCount(firstSession.refreshToken())).isZero();
    assertThat(activeTokenCount(secondSession.refreshToken())).isZero();
  }

  @Test
  void logoutAllExpiresAuthCookies() {
    var email = "logout-all-cookies@example.com";
    var password = "password123";

    createUser(email, password);
    var session = login(email, password);

    var response = authenticated(session).post("/auth/logout-all", null);

    assertThat(response.status()).isEqualTo(204);

    var setCookies = response.setCookies();

    assertThat(setCookies)
        .anyMatch(header -> header.startsWith("access_token=") && header.contains("Max-Age=0"));
    assertThat(setCookies)
        .anyMatch(header -> header.startsWith("refresh_token=") && header.contains("Max-Age=0"));
  }

  @Test
  void logoutAllDoesNotAffectOtherUsers() {
    var email = "logout-all-owner@example.com";
    var otherEmail = "logout-all-other@example.com";
    var password = "password123";

    createUser(email, password);
    createUser(otherEmail, password);

    var session = login(email, password);
    var otherSession = login(otherEmail, password);

    var response = authenticated(session).post("/auth/logout-all", null);

    assertThat(response.status()).isEqualTo(204);

    assertThat(activeTokenCount(otherSession.refreshToken())).isEqualTo(1);
  }

  @Test
  void logoutAllWithoutAccessTokenReturnsUnauthorized() {
    var response = restTemplate.postForEntity("/auth/logout-all", null, String.class);

    assertThat(response.getStatusCode().value()).isEqualTo(401);
  }

  private int activeTokenCount(String rawRefreshToken) {
    return jdbcTemplate.queryForObject(
        "SELECT COUNT(*) FROM tb_refresh_tokens WHERE token_hash = ? AND revoked_at IS NULL",
        Integer.class,
        RefreshTokenHash.sha256(rawRefreshToken));
  }
}
