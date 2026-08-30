package com.marcos.meudinheiro.identity;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import org.junit.jupiter.api.Test;
import org.springframework.http.HttpHeaders;

import com.marcos.meudinheiro.identity.infraestructure.security.token.RefreshTokenHash;

class LogoutAllIT extends AuthenticationTestSupport {

    @Test
    void logoutAllRevokesEverySessionOfCurrentUser() throws Exception {
        var email = "logout-all@example.com";
        var password = "password123";

        createUser(email, password);

        var firstSession = login(email, password);
        var secondSession = login(email, password);

        mockMvc.perform(post("/auth/logout-all")
                        .cookie(firstSession.accessTokenCookie()))
                .andExpect(status().isNoContent());

        assertThat(activeTokenCount(firstSession.refreshToken())).isZero();
        assertThat(activeTokenCount(secondSession.refreshToken())).isZero();
    }

    @Test
    void logoutAllExpiresAuthCookies() throws Exception {
        var email = "logout-all-cookies@example.com";
        var password = "password123";

        createUser(email, password);
        var session = login(email, password);

        var result = mockMvc.perform(post("/auth/logout-all")
                        .cookie(session.accessTokenCookie()))
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
    }

    @Test
    void logoutAllDoesNotAffectOtherUsers() throws Exception {
        var email = "logout-all-owner@example.com";
        var otherEmail = "logout-all-other@example.com";
        var password = "password123";

        createUser(email, password);
        createUser(otherEmail, password);

        var session = login(email, password);
        var otherSession = login(otherEmail, password);

        mockMvc.perform(post("/auth/logout-all")
                        .cookie(session.accessTokenCookie()))
                .andExpect(status().isNoContent());

        assertThat(activeTokenCount(otherSession.refreshToken()))
                .isEqualTo(1);
    }

    @Test
    void logoutAllWithoutAccessTokenReturnsUnauthorized() throws Exception {
        mockMvc.perform(post("/auth/logout-all"))
                .andExpect(status().isUnauthorized());
    }

    private int activeTokenCount(String rawRefreshToken) {
        return jdbcTemplate.queryForObject(
                "SELECT COUNT(*) FROM tb_refresh_tokens WHERE token_hash = ? AND revoked_at IS NULL",
                Integer.class,
                RefreshTokenHash.sha256(rawRefreshToken)
        );
    }
}
