package com.marcos.meudinheiro.identity.infraestructure.security.authentication;

import java.util.UUID;

import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.security.oauth2.server.resource.authentication.JwtAuthenticationToken;
import org.springframework.stereotype.Component;

import com.marcos.meudinheiro.identity.application.contract.CurrentIdentity;

@Component
final class SpringSecurityCurrentIdentity implements CurrentIdentity {

    @Override
    public UUID findCurrentAuthenticadedUser() {
        var authentication = SecurityContextHolder
            .getContext()
            .getAuthentication();

        if (!(authentication instanceof JwtAuthenticationToken jwtAuthenticationToken)) {
            throw new IllegalArgumentException("Usuario nao autenticado!");
            
        }

        return UUID.fromString(
            jwtAuthenticationToken
                .getToken()
                .getSubject()
        );
    }
}
