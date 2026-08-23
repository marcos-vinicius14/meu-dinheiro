package com.marcos.meudinheiro.identity.infraestructure.security;

import com.marcos.meudinheiro.identity.domain.model.AuthenticadedUser;
import org.springframework.security.core.Authentication;
import org.springframework.security.oauth2.jwt.JwtClaimsSet;
import org.springframework.security.oauth2.jwt.JwtEncoder;
import org.springframework.security.oauth2.jwt.JwtEncoderParameters;
import org.springframework.stereotype.Service;

import java.time.Clock;
import java.time.Duration;

@Service
public final class JwtTokenService {
    public static final String ISSUER = "meudinheiro";
    private static final Duration ACCESS_TOKEN_TTL = Duration.ofMinutes(10);

    private final JwtEncoder jwtEncoder;
    private final Clock clock;

    public JwtTokenService(JwtEncoder jwtEncoder, Clock clock) {
        this.jwtEncoder = jwtEncoder;
        this.clock = clock;
    }

    public Token generate(Authentication authentication) {
        var principal = getPrincipal(authentication);

        var issuedAt = clock.instant();
        var expiresAt = issuedAt.plus(ACCESS_TOKEN_TTL);

        var claims = JwtClaimsSet.builder()
                .issuer(ISSUER)
                .subject(principal.id().toString())
                .claim("email", principal.email())
                .issuedAt(issuedAt)
                .expiresAt(expiresAt)
                .build();

        var jwt = jwtEncoder.encode(
                JwtEncoderParameters.from(claims)
        );

        return new Token(
                jwt.getTokenValue(),
                ACCESS_TOKEN_TTL.toSeconds()
        );
    }

    private AuthenticadedUser getPrincipal(
            Authentication authentication
    ) {
        if (!(authentication.getPrincipal()
                instanceof AuthenticadedUser principal)) {

            throw new IllegalStateException(
                    "Nao foi possivel encontrar o usuario autenticado"
            );
        }

        return principal;
    }

    public record Token(
            String value,
            long expiresIn
    ) {
    }

}
