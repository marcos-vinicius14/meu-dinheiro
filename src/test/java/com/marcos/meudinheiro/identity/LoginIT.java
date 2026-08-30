package com.marcos.meudinheiro.identity;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.jdbc.core.JdbcTemplate;

import com.marcos.meudinheiro.identity.infraestructure.security.token.RefreshTokenHash;

class LoginIT extends AuthenticationTestSupport {

    @Autowired
    private JdbcTemplate jdbcTemplate;

    @Test
    void validCredentialsReturnNoContentAndSetAuthCookies() throws Exception {
        var email = "login-valid@example.com";
        var password = "password123";

        createUser(email, password);

        var result = mockMvc.perform(post("/auth/login")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "email": "%s",
                                  "password": "%s"
                                }
                                """.formatted(email, password)))
                .andExpect(status().isNoContent())
                .andReturn();

        var setCookies = result.getResponse()
                .getHeaders(HttpHeaders.SET_COOKIE);

        assertThat(setCookies)
                .anySatisfy(header -> {
                    assertThat(header).startsWith("access_token=");
                    assertThat(header).contains("HttpOnly", "Secure", "SameSite=Strict", "Path=/", "Max-Age=");
                });

        assertThat(setCookies)
                .anySatisfy(header -> {
                    assertThat(header).startsWith("refresh_token=");
                    assertThat(header).contains("HttpOnly", "Secure", "SameSite=Strict", "Path=/auth", "Max-Age=");
                });
    }

    @Test
    void validCredentialsPersistActiveRefreshTokenHash() throws Exception {
        var email = "login-persist@example.com";
        var password = "password123";

        createUser(email, password);
        var session = login(email, password);

        var tokenHash = RefreshTokenHash.sha256(session.refreshToken());

        var activeTokens = jdbcTemplate.queryForObject(
                "SELECT COUNT(*) FROM tb_refresh_tokens WHERE token_hash = ? AND revoked_at IS NULL",
                Integer.class,
                tokenHash
        );

        assertThat(activeTokens).isEqualTo(1);
    }

    @Test
    void wrongPasswordReturnsUnauthorized() throws Exception {
        var email = "login-wrong-password@example.com";
        var password = "password123";

        createUser(email, password);

        mockMvc.perform(post("/auth/login")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "email": "%s",
                                  "password": "wrong-password"
                                }
                                """.formatted(email)))
                .andExpect(status().isUnauthorized());
    }

    @Test
    void unknownEmailReturnsUnauthorized() throws Exception {
        mockMvc.perform(post("/auth/login")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "email": "does-not-exist@example.com",
                                  "password": "password123"
                                }
                                """))
                .andExpect(status().isUnauthorized());
    }

    @Test
    void malformedEmailReturnsBadRequest() throws Exception {
        mockMvc.perform(post("/auth/login")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "email": "not-an-email",
                                  "password": "password123"
                                }
                                """))
                .andExpect(status().isBadRequest());
    }

    @Test
    void shortPasswordReturnsBadRequest() throws Exception {
        mockMvc.perform(post("/auth/login")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "email": "login-short-pw@example.com",
                                  "password": "123"
                                }
                                """))
                .andExpect(status().isBadRequest());
    }

    @Test
    void uppercaseEmailLoginSucceeds() throws Exception {
        var email = "login-uppercase@example.com";
        var password = "password123";

        createUser(email, password);

        login(email.toUpperCase(), password);
    }
}
