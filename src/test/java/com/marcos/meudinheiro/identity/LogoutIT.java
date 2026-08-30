package com.marcos.meudinheiro.identity;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpHeaders;

import com.marcos.meudinheiro.identity.application.contract.RefreshTokenUseCase;
import com.marcos.meudinheiro.identity.infraestructure.security.token.RefreshTokenHash;

import jakarta.servlet.http.Cookie;

class LogoutIT extends AuthenticationTestSupport {

    @Autowired
    private RefreshTokenUseCase refreshTokenUseCase;

    @Test
    void logoutRevokesCurrentSessionAndExpiresCookies() throws Exception {
        var email = "logout-revoke@example.com";
        var password = "password123";

        createUser(email, password);
        var session = login(email, password);

        var result = mockMvc.perform(post("/auth/logout")
                        .cookie(session.refreshTokenCookie()))
                .andExpect(status().isNoContent())
                .andReturn();

        var setCookies = result.getResponse()
                .getHeaders(HttpHeaders.SET_COOKIE);

        assertThat(setCookies)
                .anyMatch(header -> header.startsWith("access_token=")
                        && header.contains("Max-Age=0"));
        assertThat(setCookies)
                .anyMatch(header -> header.startsWith("refresh_token=")
                        && header.contains("Max-Age=0"));

        var refreshResult = refreshTokenUseCase.execute(session.refreshToken());

        assertThat(refreshResult.isFailure()).isTrue();
        assertThat(refreshResult.errors()).isNotEmpty();
    }

    @Test
    void logoutWithoutCookieStillSucceeds() throws Exception {
        mockMvc.perform(post("/auth/logout"))
                .andExpect(status().isNoContent());
    }

    @Test
    void logoutWithInvalidTokenStillSucceeds() throws Exception {
        mockMvc.perform(post("/auth/logout")
                        .cookie(new Cookie(REFRESH_TOKEN_COOKIE, "not-a-real-token")))
                .andExpect(status().isNoContent());
    }

    @Test
    void logoutDoesNotRevokeOtherSessions() throws Exception {
        var email = "logout-multi-session@example.com";
        var password = "password123";

        createUser(email, password);

        var firstSession = login(email, password);
        var secondSession = login(email, password);

        mockMvc.perform(post("/auth/logout")
                        .cookie(firstSession.refreshTokenCookie()))
                .andExpect(status().isNoContent());

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
                RefreshTokenHash.sha256(rawRefreshToken)
        );
    }

    @Test
    void logoutTwiceWithSameCookieIsIdempotent() throws Exception {
        var email = "logout-idempotent@example.com";
        var password = "password123";

        createUser(email, password);
        var session = login(email, password);

        mockMvc.perform(post("/auth/logout")
                        .cookie(session.refreshTokenCookie()))
                .andExpect(status().isNoContent());

        mockMvc.perform(post("/auth/logout")
                        .cookie(session.refreshTokenCookie()))
                .andExpect(status().isNoContent());
    }
}
