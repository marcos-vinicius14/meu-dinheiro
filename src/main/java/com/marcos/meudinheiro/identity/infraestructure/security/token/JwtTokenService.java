package com.marcos.meudinheiro.identity.infraestructure.security.token;

import com.marcos.meudinheiro.identity.infraestructure.security.authentication.JwtSubject;
import java.time.Clock;
import java.time.Duration;
import org.springframework.security.oauth2.jwt.JwtClaimsSet;
import org.springframework.security.oauth2.jwt.JwtEncoder;
import org.springframework.security.oauth2.jwt.JwtEncoderParameters;
import org.springframework.stereotype.Service;

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

  public Token generate(JwtSubject subject) {

    var issuedAt = clock.instant();
    var expiresAt = issuedAt.plus(ACCESS_TOKEN_TTL);

    var claims =
        JwtClaimsSet.builder()
            .issuer(ISSUER)
            .subject(subject.userId().toString())
            .claim("email", subject.email())
            .issuedAt(issuedAt)
            .expiresAt(expiresAt)
            .build();

    var jwt = jwtEncoder.encode(JwtEncoderParameters.from(claims));

    return new Token(jwt.getTokenValue(), ACCESS_TOKEN_TTL.toSeconds());
  }

  public record Token(String value, long expiresIn) {}
}
