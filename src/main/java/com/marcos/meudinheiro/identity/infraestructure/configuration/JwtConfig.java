package com.marcos.meudinheiro.identity.infraestructure.configuration;

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
import org.springframework.security.oauth2.jwt.JwtValidators;
import org.springframework.security.oauth2.jwt.NimbusJwtDecoder;


@Configuration
public class JwtConfig {

    private static final String ISSUER = "meudinheiro";


    @Bean
    RSAPublicKey rsaPublicKey(
        @Value("${security.jwt.public.key}")
        Resource resource
    ) throws IOException {

        try (var inputStream = resource.getInputStream()) {
            return RsaKeyConverters
              .x509()
              .convert(inputStream);
        }
    }

    @Bean
    RSAPrivateKey jwPrivateKey(
        @Value("${securitty.jwt.private-key")
        Resource resource
    ) throws IOException {
        
        try (var inputStream = resource.getInputStream()) {
            return RsaKeyConverters
                .pkcs8()
                .convert(inputStream);
        }
    }


    @Bean
    JwtDecoder jwtDecoder(
        RSAPublicKey publicKey
    ) {
        var decoder = NimbusJwtDecoder
            .withPublicKey(publicKey)
            .build();

        decoder.setJwtValidator(
            JwtValidators.createDefaultWithIssuer(ISSUER)
        );

        return decoder;
    }

    @Bean
    Clock clock() {
        return Clock.systemUTC();
    }



}
