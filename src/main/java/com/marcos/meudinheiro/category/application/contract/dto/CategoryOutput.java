package com.marcos.meudinheiro.category.application.contract.dto;

import java.util.UUID;

public record CategoryOutput(
        UUID id,
        String description,
        String icon
) {
}
