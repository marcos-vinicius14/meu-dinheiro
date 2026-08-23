package com.marcos.meudinheiro.identity.infraestructure.configuration;


import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.authentication.AuthenticationManager;
import org.springframework.security.authentication.dao.DaoAuthenticationProvider;
import org.springframework.security.authentication.ProviderManager;
import org.springframework.security.crypto.bcrypt.BCryptPasswordEncoder;
import org.springframework.security.crypto.password.PasswordEncoder;

import com.marcos.meudinheiro.identity.application.usecase.SecurityUserDetailsCase;

@Configuration
public class AuthenticationConfig {

    @Bean
    PasswordEncoder passwordEncoder() {
        return new BCryptPasswordEncoder();
    }

    @Bean
    AuthenticationManager authenticationManager(
        SecurityUserDetailsCase securityUserDetailsCase,
        PasswordEncoder passwordEncoder
    ) {
        var provider = new DaoAuthenticationProvider(securityUserDetailsCase);

        provider.setPasswordEncoder(passwordEncoder);

        return new ProviderManager(provider);
    }
    
}
