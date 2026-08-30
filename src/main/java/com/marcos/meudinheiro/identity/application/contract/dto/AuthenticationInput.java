package com.marcos.meudinheiro.identity.application.contract.dto;

public record AuthenticationInput(
        String email,
        String password
) {
}
