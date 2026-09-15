package com.marcos.meudinheiro.user.application.contract.dto;

import java.util.UUID;

public record UserIdentityOutput(UUID id, String email, String passwordHash) {}
