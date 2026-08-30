package com.marcos.meudinheiro.identity.infraestructure.configuration;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.config.Customizer;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.http.SessionCreationPolicy;
import org.springframework.security.web.SecurityFilterChain;

import com.marcos.meudinheiro.identity.infraestructure.security.CookieBearerTokenResolver;
import com.marcos.meudinheiro.identity.infraestructure.security.authentication.JsonAuthenticationEntryPoint;

@Configuration
public class SecurityConfig {

    @Bean
    SecurityFilterChain securityFilterChain(
            HttpSecurity http,
            CookieBearerTokenResolver cookieBearerTokenResolver,
            JsonAuthenticationEntryPoint authenticationEntryPoint
    ) throws Exception {

        return http
            .csrf(csrf -> csrf.disable())
            .sessionManagement(session ->
                session.sessionCreationPolicy(SessionCreationPolicy.STATELESS)
            )
            .authorizeHttpRequests(auth -> auth
                .requestMatchers("/auth/login", "/auth/logout", "/auth/refresh", "/users").permitAll()
                .anyRequest().authenticated()
            )
            .exceptionHandling(handling ->
                handling.authenticationEntryPoint(authenticationEntryPoint)
            )
            .oauth2ResourceServer(resourceServer ->
                resourceServer.bearerTokenResolver(cookieBearerTokenResolver)
                    .jwt(Customizer.withDefaults())
            )
            .build();


    }

}
