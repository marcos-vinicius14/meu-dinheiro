package com.marcos.meudinheiro.identity;

import static org.assertj.core.api.Assertions.assertThat;

import com.marcos.meudinheiro.IntegrationTestSupport;
import com.marcos.meudinheiro.identity.application.contract.RefreshTokenUseCase;
import com.marcos.meudinheiro.identity.infraestructure.security.token.RefreshTokenHash;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpEntity;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpMethod;

class LogoutIT extends IntegrationTestSupport {

  @Autowired private RefreshTokenUseCase refreshTokenUseCase;

  @Test
  void logoutRevokesCurrentSessionAndExpiresCookies() {
    var email = "logout-revoke@example.com";
    var password = "password123";

    createUser(email, password);
    var session = login(email, password);

    var response =
        exchangeWithCookie(
            "/auth/logout", HttpMethod.POST, REFRESH_TOKEN_COOKIE, session.refreshToken());

    assertThat(response.getStatusCode().value()).isEqualTo(204);

    var setCookies = response.getHeaders().get(HttpHeaders.SET_COOKIE);

    assertThat(setCookies)
        .anyMatch(header -> header.startsWith("access_token=") && header.contains("Max-Age=0"));
    assertThat(setCookies)
        .anyMatch(header -> header.startsWith("refresh_token=") && header.contains("Max-Age=0"));

    var refreshResult = refreshTokenUseCase.execute(session.refreshToken());

    assertThat(refreshResult.isFailure()).isTrue();
    assertThat(refreshResult.errors()).isNotEmpty();
  }

  @Test
  void logoutWithoutCookieStillSucceeds() {
    var response = restTemplate.postForEntity("/auth/logout", null, String.class);

    assertThat(response.getStatusCode().value()).isEqualTo(204);
  }

  @Test
  void logoutWithInvalidTokenStillSucceeds() {
    var response =
        exchangeWithCookie(
            "/auth/logout", HttpMethod.POST, REFRESH_TOKEN_COOKIE, "not-a-real-token");

    assertThat(response.getStatusCode().value()).isEqualTo(204);
  }

  @Test
  void logoutDoesNotRevokeOtherSessions() {
    var email = "logout-multi-session@example.com";
    var password = "password123";

    createUser(email, password);

    var firstSession = login(email, password);
    var secondSession = login(email, password);

    var response =
        exchangeWithCookie(
            "/auth/logout", HttpMethod.POST, REFRESH_TOKEN_COOKIE, firstSession.refreshToken());

    assertThat(response.getStatusCode().value()).isEqualTo(204);

    // Estado no banco: sessao do cookie revogada, outra sessao intacta.
    // (Nao reapresentar o token revogado ao use case: isso e reuso e,
    // por design, derruba todas as sessoes do usuario.)
    assertThat(activeTokenCount(firstSession.refreshToken())).isZero();
    assertThat(activeTokenCount(secondSession.refreshToken())).isEqualTo(1);
  }

  private int activeTokenCount(String rawRefreshToken) {
    return jdbcTemplate.queryForObject(
        "SELECT COUNT(*) FROM tb_refresh_tokens WHERE token_hash = ? AND revoked_at IS NULL",
        Integer.class,
        RefreshTokenHash.sha256(rawRefreshToken));
  }

  @Test
  void logoutTwiceWithSameCookieIsIdempotent() {
    var email = "logout-idempotent@example.com";
    var password = "password123";

    createUser(email, password);
    var session = login(email, password);

    var first =
        exchangeWithCookie(
            "/auth/logout", HttpMethod.POST, REFRESH_TOKEN_COOKIE, session.refreshToken());
    var second =
        exchangeWithCookie(
            "/auth/logout", HttpMethod.POST, REFRESH_TOKEN_COOKIE, session.refreshToken());

    assertThat(first.getStatusCode().value()).isEqualTo(204);
    assertThat(second.getStatusCode().value()).isEqualTo(204);
  }

  private org.springframework.http.ResponseEntity<String> exchangeWithCookie(
      String uri, HttpMethod method, String cookieName, String cookieValue) {
    var headers = new HttpHeaders();
    headers.add(HttpHeaders.COOKIE, cookieName + "=" + cookieValue);
    headers.add(HttpHeaders.CONTENT_TYPE, "application/json");

    return restTemplate.exchange(uri, method, new HttpEntity<>(headers), String.class);
  }
}
