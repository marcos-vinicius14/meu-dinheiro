package com.marcos.meudinheiro.user.infraesctructure.web.dto;

import jakarta.validation.constraints.Email;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Size;

public record CreateUserRequest(
    @NotBlank @Size(min = 3, max = 255) String username,
    @NotBlank
        @Size(min = 3, max = 255, message = "Insira um email valido")
        @Email(message = "Insira um email valido")
        String email,
    @NotBlank @Size(min = 8, max = 255, message = "A senha deve ter no minimo 8 caracteres")
        String password) {}
