package com.marcos.meudinheiro.category.domain.valueobject;

import static org.assertj.core.api.Assertions.assertThat;

import org.junit.jupiter.api.Test;

class CategoryIconTest {

    @Test
    void validIconReturnsValue() {
        var result = CategoryIcon.create("food");

        assertThat(result.isValid()).isTrue();
        assertThat(result.value().value()).isEqualTo("food");
    }

    @Test
    void iconWithSpacesIsTrimmed() {
        var result = CategoryIcon.create("  food  ");

        assertThat(result.value().value()).isEqualTo("food");
    }

    @Test
    void nullIconIsValidWithoutValue() {
        var result = CategoryIcon.create(null);

        assertThat(result.isValid()).isTrue();
        assertThat(result.value()).isNull();
    }

    @Test
    void blankIconIsValidWithoutValue() {
        var result = CategoryIcon.create("   ");

        assertThat(result.isValid()).isTrue();
        assertThat(result.value()).isNull();
    }

    @Test
    void longIconIsInvalid() {
        var result = CategoryIcon.create("A".repeat(256));

        assertThat(result.isInvalid()).isTrue();
        assertThat(result.errors()).containsExactly("Ícone da categoria deve ter no máximo 255 caracteres");
    }
}
