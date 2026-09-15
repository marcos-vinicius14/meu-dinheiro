package com.marcos.meudinheiro.identity;

import static org.assertj.core.api.Assertions.assertThat;

import com.marcos.meudinheiro.IntegrationTestSupport;
import com.marcos.meudinheiro.identity.infraestructure.security.token.JwtTokenService;
import com.nimbusds.jose.jwk.JWKSet;
import com.nimbusds.jose.jwk.RSAKey;
import com.nimbusds.jose.jwk.source.ImmutableJWKSet;
import java.security.KeyPair;
import java.security.KeyPairGenerator;
import java.security.NoSuchAlgorithmException;
import java.security.interfaces.RSAPrivateKey;
import java.security.interfaces.RSAPublicKey;
import java.time.Instant;
import java.util.UUID;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.oauth2.jwt.JwtClaimsSet;
import org.springframework.security.oauth2.jwt.JwtEncoderParameters;
import org.springframework.security.oauth2.jwt.NimbusJwtEncoder;

class CookieAuthenticationIT extends IntegrationTestSupport {

  @Autowired private RSAPublicKey publicKey;

  @Autowired private RSAPrivateKey privateKey;

  @Test
  void validAccessTokenCookieGrantsAccessToProtectedEndpoint() {
    var email = "cookie-valid@example.com";
    var password = "password123";

    createUser(email, password);
    var session = login(email, password);

    var response = authenticated(session).get("/auth/me");

    assertThat(response.status()).isEqualTo(200);
    assertThat(response.body()).contains("\"email\":\"" + email + "\"");
  }

  @Test
  void missingAccessTokenCookieReturnsUnauthorized() {
    var response = restTemplate.getForEntity("/auth/me", String.class);

    assertThat(response.getStatusCode().value()).isEqualTo(401);
    assertThat(response.getBody()).contains("errors");
  }

  @Test
  void garbageAccessTokenCookieReturnsUnauthorized() {
    var response = withCookie(ACCESS_TOKEN_COOKIE, "garbage-token").get("/auth/me");

    assertThat(response.status()).isEqualTo(401);
  }

  @Test
  void tokenSignedWithUnknownKeyIsRejected() throws Exception {
    var forgedKeyPair = generateRsaKeyPair();

    var token =
        encodeToken(
            (RSAPublicKey) forgedKeyPair.getPublic(),
            (RSAPrivateKey) forgedKeyPair.getPrivate(),
            Instant.now().plusSeconds(600));

    var response = withCookie(ACCESS_TOKEN_COOKIE, token).get("/auth/me");

    assertThat(response.status()).isEqualTo(401);
  }

  @Test
  void expiredTokenIsRejected() {
    var now = Instant.now();

    var token = encodeToken(publicKey, privateKey, now.minusSeconds(300));

    var response = withCookie(ACCESS_TOKEN_COOKIE, token).get("/auth/me");

    assertThat(response.status()).isEqualTo(401);
  }

  private String encodeToken(
      RSAPublicKey signingPublicKey, RSAPrivateKey signingPrivateKey, Instant expiresAt) {
    var jwk = new RSAKey.Builder(signingPublicKey).privateKey(signingPrivateKey).build();

    var encoder = new NimbusJwtEncoder(new ImmutableJWKSet<>(new JWKSet(jwk)));

    var claims =
        JwtClaimsSet.builder()
            .issuer(JwtTokenService.ISSUER)
            .subject(UUID.randomUUID().toString())
            .claim("email", "forged@example.com")
            .issuedAt(expiresAt.minusSeconds(600))
            .expiresAt(expiresAt)
            .build();

    return encoder.encode(JwtEncoderParameters.from(claims)).getTokenValue();
  }

  private KeyPair generateRsaKeyPair() throws NoSuchAlgorithmException {
    var generator = KeyPairGenerator.getInstance("RSA");
    generator.initialize(2048);
    return generator.generateKeyPair();
  }
}
