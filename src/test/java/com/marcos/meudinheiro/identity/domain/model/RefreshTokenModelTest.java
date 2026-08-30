package com.marcos.meudinheiro.identity.domain.model;

import static org.assertj.core.api.Assertions.assertThat;

import java.time.Instant;
import java.util.UUID;

import org.junit.jupiter.api.Test;

class RefreshTokenModelTest {

    private static final Instant NOW = Instant.parse("2026-01-15T12:00:00Z");
    private static final Instant FUTURE = NOW.plusSeconds(3600);
    private static final Instant PAST = NOW.minusSeconds(3600);

    @Test
    void freshTokenIsValid() {
        var token = createToken(FUTURE);

        assertThat(token.isValid(NOW)).isTrue();
        assertThat(token.isRevoked()).isFalse();
        assertThat(token.isExpired(NOW)).isFalse();
    }

    @Test
    void expiredTokenIsInvalid() {
        var token = createToken(PAST);

        assertThat(token.isExpired(NOW)).isTrue();
        assertThat(token.isValid(NOW)).isFalse();
    }

    @Test
    void tokenExpiringExactlyNowIsExpired() {
        var token = createToken(NOW);

        assertThat(token.isExpired(NOW)).isTrue();
    }

    @Test
    void revokeMarksTokenAsRevokedAndInvalid() {
        var token = createToken(FUTURE);

        token.revoke(NOW);

        assertThat(token.isRevoked()).isTrue();
        assertThat(token.isValid(NOW)).isFalse();
    }

    @Test
    void revokeIsIdempotent() {
        var token = createToken(FUTURE);

        token.revoke(NOW);
        token.revoke(NOW.plusSeconds(60));

        assertThat(token.isRevoked()).isTrue();
        assertThat(token.isValid(NOW.plusSeconds(60))).isFalse();
    }

    @Test
    void tokenCarriesUserIdAndHash() {
        var userId = UUID.randomUUID();

        var token = RefreshTokenModel.create(
                userId,
                "abc123",
                FUTURE,
                NOW
        );

        assertThat(token.getUserId()).isEqualTo(userId);
        assertThat(token.getTokenHash()).isEqualTo("abc123");
    }

    private RefreshTokenModel createToken(Instant expiresAt) {
        return RefreshTokenModel.create(
                UUID.randomUUID(),
                "hash",
                expiresAt,
                NOW
        );
    }
}
