package com.marcos.meudinheiro.category.infraestructure.web.dto;

import com.marcos.meudinheiro.category.application.contract.dto.CategoryOutput;

import java.util.UUID;

public record CategoryResponse(
        UUID id,
        String description,
        String icon,
        boolean isFlexible
) {

    public static CategoryResponse from(CategoryOutput output) {
        return new CategoryResponse(
                output.id(),
                output.description(),
                output.icon(),
                output.isFlexible()
        );
    }
}
