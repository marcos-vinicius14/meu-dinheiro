package com.marcos.meudinheiro.identity;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import org.junit.jupiter.api.Test;
import org.springframework.http.HttpHeaders;
import org.springframework.test.web.servlet.MvcResult;

import com.marcos.meudinheiro.identity.infraestructure.security.token.RefreshTokenHash;

import jakarta.servlet.http.Cookie;

class RefreshEndpointIT extends AuthenticationTestSupport {

    @Test
    void validRefreshTokenCookieReturnsNewSessionCookies() throws Exception {
        var email = "refresh-endpoint-valid@example.com";
        var password = "password123";

        createUser(email, password);
        var session = login(email, password);

        var result = mockMvc.perform(post("/auth/refresh")
                        .cookie(session.refreshTokenCookie()))
                .andExpect(status().isNoContent())
                .andReturn();

        var newAccessToken = cookieValue(result, ACCESS_TOKEN_COOKIE);
        var newRefreshToken = cookieValue(result, REFRESH_TOKEN_COOKIE);

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
    void oldRefreshTokenIsRevokedAfterRotation() throws Exception {
        var email = "refresh-endpoint-rotation@example.com";
        var password = "password123";

        createUser(email, password);
        var session = login(email, password);

        mockMvc.perform(post("/auth/refresh")
                        .cookie(session.refreshTokenCookie()))
                .andExpect(status().isNoContent());

        var revokedTokens = jdbcTemplate.queryForObject(
                "SELECT COUNT(*) FROM tb_refresh_tokens WHERE token_hash = ? AND revoked_at IS NOT NULL",
                Integer.class,
                RefreshTokenHash.sha256(session.refreshToken())
        );

        assertThat(revokedTokens).isEqualTo(1);
    }

    @Test
    void revokedRefreshTokenReturnsUnauthorizedAndClearsCookies() throws Exception {
        var email = "refresh-endpoint-revoked@example.com";
        var password = "password123";

        createUser(email, password);
        var session = login(email, password);

        mockMvc.perform(post("/auth/refresh")
                        .cookie(session.refreshTokenCookie()))
                .andExpect(status().isNoContent());

        var result = mockMvc.perform(post("/auth/refresh")
                        .cookie(session.refreshTokenCookie()))
                .andExpect(status().isUnauthorized())
                .andExpect(jsonPath("$.errors").isArray())
                .andReturn();

        assertCookiesExpired(result);
    }

    @Test
    void missingRefreshTokenCookieReturnsUnauthorized() throws Exception {
        var result = mockMvc.perform(post("/auth/refresh"))
                .andExpect(status().isUnauthorized())
                .andReturn();

        assertCookiesExpired(result);
    }

    @Test
    void garbageRefreshTokenReturnsUnauthorized() throws Exception {
        mockMvc.perform(post("/auth/refresh")
                        .cookie(new Cookie(REFRESH_TOKEN_COOKIE, "garbage-token")))
                .andExpect(status().isUnauthorized());
    }

    private void assertCookiesExpired(MvcResult result) {
        var setCookies = result.getResponse()
                .getHeaders(HttpHeaders.SET_COOKIE);

        assertThat(setCookies)
                .anyMatch(header -> header.startsWith("access_token=")
                        && header.contains("Max-Age=0"));
        assertThat(setCookies)
                .anyMatch(header -> header.startsWith("refresh_token=")
                        && header.contains("Max-Age=0"));
    }
}
