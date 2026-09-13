package com.marcos.meudinheiro.category.infraestructure.web.dto;

import com.marcos.meudinheiro.category.domain.valueobject.CategoryDescription;
import com.marcos.meudinheiro.category.domain.valueobject.CategoryIcon;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Size;

public record CreateCategoryRequest(
                @NotBlank(message = CategoryDescription.DESCRIPTION_REQUIRED) String description,

                @Size(max = 255, message = CategoryIcon.ICON_SIZE) String icon) {
}
