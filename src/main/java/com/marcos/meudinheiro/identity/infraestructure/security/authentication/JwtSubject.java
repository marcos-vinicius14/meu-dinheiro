package com.marcos.meudinheiro.identity.infraestructure.security.authentication;

import java.util.UUID;

public record JwtSubject(UUID userId, String email) {}
