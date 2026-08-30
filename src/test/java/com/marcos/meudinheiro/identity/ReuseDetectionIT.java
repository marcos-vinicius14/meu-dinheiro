package com.marcos.meudinheiro.identity;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import org.junit.jupiter.api.Test;

import com.marcos.meudinheiro.identity.infraestructure.security.token.RefreshTokenHash;

class ReuseDetectionIT extends AuthenticationTestSupport {

    @Test
    void reusingRotatedTokenRevokesAllUserSessions() throws Exception {
        var email = "reuse-revoke-all@example.com";
        var password = "password123";

        createUser(email, password);
        var firstSession = login(email, password);
        var secondSession = login(email, password);

        // Rotacao legitima: o token antigo da primeira sessao morre
        mockMvc.perform(post("/auth/refresh")
                        .cookie(firstSession.refreshTokenCookie()))
                .andExpect(status().isNoContent());

        // Reuso do token ja rotacionado = roubo presumido
        mockMvc.perform(post("/auth/refresh")
                        .cookie(firstSession.refreshTokenCookie()))
                .andExpect(status().isUnauthorized());

        // Colateral: TODAS as sessoes do usuario morrem
        assertThat(activeTokenCount(secondSession.refreshToken()))
                .isZero();
    }

    @Test
    void reuseDetectionDoesNotAffectOtherUsers() throws Exception {
        var victimEmail = "reuse-victim@example.com";
        var bystanderEmail = "reuse-bystander@example.com";
        var password = "password123";

        createUser(victimEmail, password);
        createUser(bystanderEmail, password);

        var victimSession = login(victimEmail, password);
        var bystanderSession = login(bystanderEmail, password);

        mockMvc.perform(post("/auth/refresh")
                        .cookie(victimSession.refreshTokenCookie()))
                .andExpect(status().isNoContent());

        mockMvc.perform(post("/auth/refresh")
                        .cookie(victimSession.refreshTokenCookie()))
                .andExpect(status().isUnauthorized());

        assertThat(activeTokenCount(bystanderSession.refreshToken()))
                .isEqualTo(1);
    }

    @Test
    void expiredTokenDoesNotTriggerReuseDetection() throws Exception {
        var email = "reuse-expired@example.com";
        var password = "password123";

        createUser(email, password);
        var firstSession = login(email, password);
        var secondSession = login(email, password);

        // Token expirado por idade (nao revogado): falha normal, sem punicao
        jdbcTemplate.update(
                "UPDATE tb_refresh_tokens SET expires_at = CURRENT_TIMESTAMP - INTERVAL '1 hour' WHERE token_hash = ?",
                RefreshTokenHash.sha256(firstSession.refreshToken())
        );

        mockMvc.perform(post("/auth/refresh")
                        .cookie(firstSession.refreshTokenCookie()))
                .andExpect(status().isUnauthorized());

        // A outra sessao sobrevive: expiracao nao e roubo
        assertThat(activeTokenCount(secondSession.refreshToken()))
                .isEqualTo(1);
    }

    private int activeTokenCount(String rawRefreshToken) {
        return jdbcTemplate.queryForObject(
                "SELECT COUNT(*) FROM tb_refresh_tokens WHERE token_hash = ? AND revoked_at IS NULL",
                Integer.class,
                RefreshTokenHash.sha256(rawRefreshToken)
        );
    }
}
