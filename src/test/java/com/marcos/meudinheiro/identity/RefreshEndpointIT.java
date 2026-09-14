package com.marcos.meudinheiro.identity;

import static org.assertj.core.api.Assertions.assertThat;

import org.junit.jupiter.api.Test;
import org.springframework.http.HttpEntity;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpMethod;

import com.marcos.meudinheiro.IntegrationTestSupport;
import com.marcos.meudinheiro.identity.infraestructure.security.token.RefreshTokenHash;

class RefreshEndpointIT extends IntegrationTestSupport {

    @Test
    void validRefreshTokenCookieReturnsNewSessionCookies() {
        var email = "refresh-endpoint-valid@example.com";
        var password = "password123";

        createUser(email, password);
        var session = login(email, password);

        var response = exchangeWithCookie(session.refreshToken());

        assertThat(response.getStatusCode().value()).isEqualTo(204);

        var setCookies = response.getHeaders().get(HttpHeaders.SET_COOKIE);
        var newAccessToken = cookieFrom(setCookies, ACCESS_TOKEN_COOKIE);
        var newRefreshToken = cookieFrom(setCookies, REFRESH_TOKEN_COOKIE);

        assertThat(newAccessToken).isNotBlank();
        assertThat(newRefreshToken).isNotEqualTo(session.refreshToken());

        var tokenHash = RefreshTokenHash.sha256(newRefreshToken);

        var activeTokens = jdbcTemplate.queryForObject(
                "SELECT COUNT(*) FROM tb_refresh_tokens WHERE token_hash = ? AND revoked_at IS NULL",
                Integer.class,
                tokenHash
        );

        assertThat(activeTokens).isEqualTo(1);
    }

    @Test
    void oldRefreshTokenIsRevokedAfterRotation() {
        var email = "refresh-endpoint-rotation@example.com";
        var password = "password123";

        createUser(email, password);
        var session = login(email, password);

        var response = exchangeWithCookie(session.refreshToken());

        assertThat(response.getStatusCode().value()).isEqualTo(204);

        var revokedTokens = jdbcTemplate.queryForObject(
                "SELECT COUNT(*) FROM tb_refresh_tokens WHERE token_hash = ? AND revoked_at IS NOT NULL",
                Integer.class,
                RefreshTokenHash.sha256(session.refreshToken())
        );

        assertThat(revokedTokens).isEqualTo(1);
    }

    @Test
    void revokedRefreshTokenReturnsUnauthorizedAndClearsCookies() {
        var email = "refresh-endpoint-revoked@example.com";
        var password = "password123";

        createUser(email, password);
        var session = login(email, password);

        assertThat(exchangeWithCookie(session.refreshToken()).getStatusCode().value())
                .isEqualTo(204);

        var result = exchangeWithCookie(session.refreshToken());

        assertThat(result.getStatusCode().value()).isEqualTo(401);
        assertCookiesExpired(result.getHeaders().get(HttpHeaders.SET_COOKIE));
    }

    @Test
    void missingRefreshTokenCookieReturnsUnauthorized() {
        var response = restTemplate.postForEntity("/auth/refresh", null, String.class);

        assertThat(response.getStatusCode().value()).isEqualTo(401);
        assertCookiesExpired(response.getHeaders().get(HttpHeaders.SET_COOKIE));
    }

    @Test
    void garbageRefreshTokenReturnsUnauthorized() {
        var response = exchangeWithCookie("garbage-token");

        assertThat(response.getStatusCode().value()).isEqualTo(401);
    }

    private org.springframework.http.ResponseEntity<String> exchangeWithCookie(String refreshToken) {
        var headers = new HttpHeaders();
        headers.add(HttpHeaders.COOKIE, REFRESH_TOKEN_COOKIE + "=" + refreshToken);
        headers.add(HttpHeaders.CONTENT_TYPE, "application/json");

        return restTemplate.exchange(
                "/auth/refresh",
                HttpMethod.POST,
                new HttpEntity<>(headers),
                String.class
        );
    }

    private static String cookieFrom(java.util.List<String> setCookies, String name) {
        return setCookies.stream()
                .filter(header -> header.startsWith(name + "="))
                .map(header -> header.substring((name + "=").length()).split(";", 2)[0])
                .findFirst()
                .orElseThrow(() -> new AssertionError(name + " cookie ausente"));
    }

    private void assertCookiesExpired(java.util.List<String> setCookies) {
        assertThat(setCookies)
                .anyMatch(header -> header.startsWith("access_token=")
                        && header.contains("Max-Age=0"));
        assertThat(setCookies)
                .anyMatch(header -> header.startsWith("refresh_token=")
                        && header.contains("Max-Age=0"));
    }
}
