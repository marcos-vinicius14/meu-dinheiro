package com.marcos.meudinheiro.identity.infraestructure.configuration;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.Customizer;
import org.springframework.security.config.http.SessionCreationPolicy;
import org.springframework.security.web.SecurityFilterChain;

import com.marcos.meudinheiro.identity.infraestructure.security.CookieBearerTokenResolver;

@Configuration
public class SecurityConfig {

    @Bean
    SecurityFilterChain securityFilterChain(HttpSecurity http, CookieBearerTokenResolver cookieBearerTokenResolver) throws Exception {

        return http
            .sessionManagement(session ->
                session.sessionCreationPolicy(SessionCreationPolicy.STATELESS)
            )
            .authorizeHttpRequests(auth -> auth
                .requestMatchers("/auth/login", "/users").permitAll()
                .anyRequest().authenticated()
            )
            .oauth2ResourceServer(resourceServer ->
                resourceServer.bearerTokenResolver(cookieBearerTokenResolver)
                    .jwt(Customizer.withDefaults())
            )
            .build();

        
    }
    
}
