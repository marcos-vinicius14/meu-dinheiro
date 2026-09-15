package com.marcos.meudinheiro.identity;

import static org.assertj.core.api.Assertions.assertThat;

import com.marcos.meudinheiro.IntegrationTestSupport;
import com.marcos.meudinheiro.identity.infraestructure.security.token.RefreshTokenHash;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpHeaders;
import org.springframework.jdbc.core.JdbcTemplate;

class LoginIT extends IntegrationTestSupport {

  @Autowired private JdbcTemplate jdbcTemplate;

  @Test
  void validCredentialsReturnNoContentAndSetAuthCookies() {
    var email = "login-valid@example.com";
    var password = "password123";

    createUser(email, password);

    var response =
        restTemplate.postForEntity(
            "/auth/login",
            new org.springframework.http.HttpEntity<>(
                """
                                {
                                  "email": "%s",
                                  "password": "%s"
                                }
                                """
                    .formatted(email, password),
                jsonHeaders()),
            String.class);

    assertThat(response.getStatusCode().value()).isEqualTo(204);

    var setCookies = response.getHeaders().get(HttpHeaders.SET_COOKIE);

    assertThat(setCookies)
        .anySatisfy(
            header -> {
              assertThat(header).startsWith("access_token=");
              assertThat(header)
                  .contains("HttpOnly", "Secure", "SameSite=Strict", "Path=/", "Max-Age=");
            });

    assertThat(setCookies)
        .anySatisfy(
            header -> {
              assertThat(header).startsWith("refresh_token=");
              assertThat(header)
                  .contains("HttpOnly", "Secure", "SameSite=Strict", "Path=/auth", "Max-Age=");
            });
  }

  @Test
  void validCredentialsPersistActiveRefreshTokenHash() {
    var email = "login-persist@example.com";
    var password = "password123";

    createUser(email, password);
    var session = loginReal(email, password);

    var tokenHash = RefreshTokenHash.sha256(session.refreshToken());

    var activeTokens =
        jdbcTemplate.queryForObject(
            "SELECT COUNT(*) FROM tb_refresh_tokens WHERE token_hash = ? AND revoked_at IS NULL",
            Integer.class,
            tokenHash);

    assertThat(activeTokens).isEqualTo(1);
  }

  @Test
  void wrongPasswordReturnsUnauthorized() {
    var email = "login-wrong-password@example.com";
    var password = "password123";

    createUser(email, password);

    var response =
        restTemplate.postForEntity(
            "/auth/login",
            new org.springframework.http.HttpEntity<>(
                """
                                {
                                  "email": "%s",
                                  "password": "wrong-password"
                                }
                                """
                    .formatted(email),
                jsonHeaders()),
            String.class);

    assertThat(response.getStatusCode().value()).isEqualTo(401);
  }

  @Test
  void unknownEmailReturnsUnauthorized() {
    var response =
        restTemplate.postForEntity(
            "/auth/login",
            new org.springframework.http.HttpEntity<>(
                """
                                {
                                  "email": "does-not-exist@example.com",
                                  "password": "password123"
                                }
                                """,
                jsonHeaders()),
            String.class);

    assertThat(response.getStatusCode().value()).isEqualTo(401);
  }

  @Test
  void malformedEmailReturnsBadRequest() {
    var response =
        restTemplate.postForEntity(
            "/auth/login",
            new org.springframework.http.HttpEntity<>(
                """
                                {
                                  "email": "not-an-email",
                                  "password": "password123"
                                }
                                """,
                jsonHeaders()),
            String.class);

    assertThat(response.getStatusCode().value()).isEqualTo(400);
  }

  @Test
  void shortPasswordReturnsBadRequest() {
    var response =
        restTemplate.postForEntity(
            "/auth/login",
            new org.springframework.http.HttpEntity<>(
                """
                                {
                                  "email": "login-short-pw@example.com",
                                  "password": "123"
                                }
                                """,
                jsonHeaders()),
            String.class);

    assertThat(response.getStatusCode().value()).isEqualTo(400);
  }

  @Test
  void uppercaseEmailLoginSucceeds() {
    var email = "login-uppercase@example.com";
    var password = "password123";

    createUser(email, password);

    var session = loginReal(email.toUpperCase(), password);

    assertThat(session.accessToken()).isNotBlank();
    assertThat(session.refreshToken()).isNotBlank();
  }
}
