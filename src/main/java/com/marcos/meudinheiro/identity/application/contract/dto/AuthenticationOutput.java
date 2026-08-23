package com.marcos.meudinheiro.identity.application.contract.dto;

public record AuthenticationOutput(
        String acessToken,
        String tokenType,
        long expiresIn
) {
    public static AuthenticationOutput bearer(
            String acessToken,
            long expiresIn
    ) {
        return new AuthenticationOutput(
                acessToken,
                "Bearer",
                expiresIn
        );
    }
}
