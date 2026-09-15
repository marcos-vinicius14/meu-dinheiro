package com.marcos.meudinheiro.identity.infraestructure.configuration;

import com.marcos.meudinheiro.identity.infraestructure.security.token.JwtTokenService;
import com.nimbusds.jose.jwk.JWKSet;
import com.nimbusds.jose.jwk.RSAKey;
import com.nimbusds.jose.jwk.source.ImmutableJWKSet;
import java.io.IOException;
import java.security.interfaces.RSAPrivateKey;
import java.security.interfaces.RSAPublicKey;
import java.time.Clock;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.io.Resource;
import org.springframework.security.converter.RsaKeyConverters;
import org.springframework.security.oauth2.jwt.JwtDecoder;
import org.springframework.security.oauth2.jwt.JwtEncoder;
import org.springframework.security.oauth2.jwt.JwtValidators;
import org.springframework.security.oauth2.jwt.NimbusJwtDecoder;
import org.springframework.security.oauth2.jwt.NimbusJwtEncoder;

@Configuration
public class JwtConfig {

  @Bean
  RSAPublicKey rsaPublicKey(@Value("${security.jwt.public-key}") Resource resource)
      throws IOException {

    try (var inputStream = resource.getInputStream()) {
      return RsaKeyConverters.x509().convert(inputStream);
    }
  }

  @Bean
  RSAPrivateKey jwtPrivateKey(@Value("${security.jwt.private-key}") Resource resource)
      throws IOException {

    try (var inputStream = resource.getInputStream()) {
      return RsaKeyConverters.pkcs8().convert(inputStream);
    }
  }

  @Bean
  JwtDecoder jwtDecoder(RSAPublicKey publicKey) {
    var decoder = NimbusJwtDecoder.withPublicKey(publicKey).build();

    decoder.setJwtValidator(JwtValidators.createDefaultWithIssuer(JwtTokenService.ISSUER));

    return decoder;
  }

  @Bean
  JwtEncoder jwtEncoder(RSAPublicKey publicKey, RSAPrivateKey privateKey) {
    var jwk = new RSAKey.Builder(publicKey).privateKey(privateKey).build();

    return new NimbusJwtEncoder(new ImmutableJWKSet<>(new JWKSet(jwk)));
  }

  @Bean
  Clock clock() {
    return Clock.systemUTC();
  }
}
