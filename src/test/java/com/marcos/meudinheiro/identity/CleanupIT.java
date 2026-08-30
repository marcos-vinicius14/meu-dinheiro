package com.marcos.meudinheiro.identity;

import static org.assertj.core.api.Assertions.assertThat;

import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;

import com.marcos.meudinheiro.identity.infraestructure.jobs.RefreshTokenCleanupService;
import com.marcos.meudinheiro.identity.infraestructure.security.token.RefreshTokenHash;

class CleanupIT extends AuthenticationTestSupport {

    @Autowired
    private RefreshTokenCleanupService cleanupService;

    @Test
    void removesTokensExpiredBeforeRetentionCutoff() throws Exception {
        var email = "cleanup-expired-old@example.com";
        var password = "password123";

        createUser(email, password);
        var session = login(email, password);

        // Expirado a 10 dias: alem da retencao de 7 dias -> removido
        jdbcTemplate.update(
                "UPDATE tb_refresh_tokens SET expires_at = CURRENT_TIMESTAMP - INTERVAL '10 days' WHERE token_hash = ?",
                RefreshTokenHash.sha256(session.refreshToken())
        );

        cleanupService.cleanupExpiredTokens();

        var remaining = jdbcTemplate.queryForObject(
                "SELECT COUNT(*) FROM tb_refresh_tokens WHERE token_hash = ?",
                Integer.class,
                RefreshTokenHash.sha256(session.refreshToken())
        );

        assertThat(remaining).isZero();
    }

    @Test
    void keepsTokensExpiredWithinRetentionWindow() throws Exception {
        var email = "cleanup-expired-recent@example.com";
        var password = "password123";

        createUser(email, password);
        var session = login(email, password);

        // Expirado a 1 dia: dentro da retencao de 7 dias -> mantido
        jdbcTemplate.update(
                "UPDATE tb_refresh_tokens SET expires_at = CURRENT_TIMESTAMP - INTERVAL '1 day' WHERE token_hash = ?",
                RefreshTokenHash.sha256(session.refreshToken())
        );

        cleanupService.cleanupExpiredTokens();

        var remaining = jdbcTemplate.queryForObject(
                "SELECT COUNT(*) FROM tb_refresh_tokens WHERE token_hash = ?",
                Integer.class,
                RefreshTokenHash.sha256(session.refreshToken())
        );

        assertThat(remaining).isEqualTo(1);
    }

    @Test
    void keepsActiveTokens() throws Exception {
        var email = "cleanup-active@example.com";
        var password = "password123";

        createUser(email, password);
        var session = login(email, password);

        cleanupService.cleanupExpiredTokens();

        var remaining = jdbcTemplate.queryForObject(
                "SELECT COUNT(*) FROM tb_refresh_tokens WHERE token_hash = ?",
                Integer.class,
                RefreshTokenHash.sha256(session.refreshToken())
        );

        assertThat(remaining).isEqualTo(1);
    }

    @Test
    void removesRevokedTokensBeyondRetention() throws Exception {
        var email = "cleanup-revoked-old@example.com";
        var password = "password123";

        createUser(email, password);
        var session = login(email, password);

        jdbcTemplate.update(
                "UPDATE tb_refresh_tokens SET revoked_at = CURRENT_TIMESTAMP - INTERVAL '10 days' WHERE token_hash = ?",
                RefreshTokenHash.sha256(session.refreshToken())
        );

        cleanupService.cleanupExpiredTokens();

        var remaining = jdbcTemplate.queryForObject(
                "SELECT COUNT(*) FROM tb_refresh_tokens WHERE token_hash = ?",
                Integer.class,
                RefreshTokenHash.sha256(session.refreshToken())
        );

        assertThat(remaining).isZero();
    }
}
