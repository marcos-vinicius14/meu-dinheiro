package com.marcos.meudinheiro.identity.infraestructure.web.dto;

import jakarta.validation.constraints.Email;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Size;

public record LoginRequest(
        @NotBlank
        @Email
        String email,

        @NotBlank
        @Size(min = 8, max = 255, message = "Quantidade de caracteres invalida")
        String password
) {
}
