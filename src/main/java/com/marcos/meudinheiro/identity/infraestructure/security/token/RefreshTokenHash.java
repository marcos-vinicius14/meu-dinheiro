package com.marcos.meudinheiro.identity.infraestructure.security.token;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.HexFormat;

public final class RefreshTokenHash {

    private RefreshTokenHash() {
    }

    public static String sha256(String token) {
        try {
            var digest = MessageDigest.getInstance("SHA-256");

            var hash = digest.digest(
                token.getBytes(StandardCharsets.UTF_8)
            );

            return HexFormat.of().formatHex(hash);

        } catch (NoSuchAlgorithmException exception) {
            throw new IllegalStateException(
                "SHA-256 unavailable",
                exception
            );
        }
    }
}