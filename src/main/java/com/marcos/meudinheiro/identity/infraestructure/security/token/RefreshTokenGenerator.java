package com.marcos.meudinheiro.identity.infraestructure.security.token;

import java.security.SecureRandom;
import java.util.Base64;

import org.springframework.stereotype.Component;

@Component
public final class RefreshTokenGenerator {
    private static final int TOKEN_SIZE_BYTES = 64;

    private final SecureRandom secureRandom = new SecureRandom();

    public GeneratedRefreshToken generate() {

        var bytes = new byte[TOKEN_SIZE_BYTES];

        secureRandom.nextBytes(bytes);

        var token = Base64.getUrlEncoder()
            .withoutPadding()
            .encodeToString(bytes);

        return new GeneratedRefreshToken(
            token,
            RefreshTokenHash.sha256(token)
        );

    }


    public record GeneratedRefreshToken(
        String value,
        String hash
    ) {
    }
}
