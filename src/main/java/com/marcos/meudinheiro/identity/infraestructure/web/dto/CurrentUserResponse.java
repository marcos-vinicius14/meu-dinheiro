package com.marcos.meudinheiro.identity.infraestructure.web.dto;

import java.util.UUID;

public record CurrentUserResponse(UUID id, String email) {}
