package com.marcos.meudinheiro.category.application.mapper;

import com.marcos.meudinheiro.category.application.contract.dto.CategoryOutput;
import com.marcos.meudinheiro.category.domain.model.CategoryModel;

public final class CategoryMapper {

    private CategoryMapper() {}

    public static CategoryOutput toOutput(CategoryModel category) {
        return new CategoryOutput(
                category.getId(),
                category.getDescription(),
                category.getIcon()
        );
    }
}
