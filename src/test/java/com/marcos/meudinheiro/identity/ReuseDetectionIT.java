package com.marcos.meudinheiro.identity;

import static org.assertj.core.api.Assertions.assertThat;

import com.marcos.meudinheiro.IntegrationTestSupport;
import com.marcos.meudinheiro.identity.infraestructure.security.token.RefreshTokenHash;
import org.junit.jupiter.api.Test;
import org.springframework.http.HttpEntity;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpMethod;

class ReuseDetectionIT extends IntegrationTestSupport {

  @Test
  void reusingRotatedTokenRevokesAllUserSessions() {
    var email = "reuse-revoke-all@example.com";
    var password = "password123";

    createUser(email, password);
    var firstSession = login(email, password);
    var secondSession = login(email, password);

    // Rotacao legitima: o token antigo da primeira sessao morre
    assertThat(refresh(firstSession.refreshToken()).getStatusCode().value()).isEqualTo(204);

    // Reuso do token ja rotacionado = roubo presumido
    assertThat(refresh(firstSession.refreshToken()).getStatusCode().value()).isEqualTo(401);

    // Colateral: TODAS as sessoes do usuario morrem
    assertThat(activeTokenCount(secondSession.refreshToken())).isZero();
  }

  @Test
  void reuseDetectionDoesNotAffectOtherUsers() {
    var victimEmail = "reuse-victim@example.com";
    var bystanderEmail = "reuse-bystander@example.com";
    var password = "password123";

    createUser(victimEmail, password);
    createUser(bystanderEmail, password);

    var victimSession = login(victimEmail, password);
    var bystanderSession = login(bystanderEmail, password);

    assertThat(refresh(victimSession.refreshToken()).getStatusCode().value()).isEqualTo(204);

    assertThat(refresh(victimSession.refreshToken()).getStatusCode().value()).isEqualTo(401);

    assertThat(activeTokenCount(bystanderSession.refreshToken())).isEqualTo(1);
  }

  @Test
  void expiredTokenDoesNotTriggerReuseDetection() {
    var email = "reuse-expired@example.com";
    var password = "password123";

    createUser(email, password);
    var firstSession = login(email, password);
    var secondSession = login(email, password);

    // Token expirado por idade (nao revogado): falha normal, sem punicao
    jdbcTemplate.update(
        "UPDATE tb_refresh_tokens SET expires_at = CURRENT_TIMESTAMP - INTERVAL '1 hour' WHERE token_hash = ?",
        RefreshTokenHash.sha256(firstSession.refreshToken()));

    assertThat(refresh(firstSession.refreshToken()).getStatusCode().value()).isEqualTo(401);

    // A outra sessao sobrevive: expiracao nao e roubo
    assertThat(activeTokenCount(secondSession.refreshToken())).isEqualTo(1);
  }

  private org.springframework.http.ResponseEntity<String> refresh(String refreshToken) {
    var headers = new HttpHeaders();
    headers.add(HttpHeaders.COOKIE, REFRESH_TOKEN_COOKIE + "=" + refreshToken);
    headers.add(HttpHeaders.CONTENT_TYPE, "application/json");

    return restTemplate.exchange(
        "/auth/refresh", HttpMethod.POST, new HttpEntity<>(headers), String.class);
  }

  private int activeTokenCount(String rawRefreshToken) {
    return jdbcTemplate.queryForObject(
        "SELECT COUNT(*) FROM tb_refresh_tokens WHERE token_hash = ? AND revoked_at IS NULL",
        Integer.class,
        RefreshTokenHash.sha256(rawRefreshToken));
  }
}
