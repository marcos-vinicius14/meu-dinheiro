package com.marcos.meudinheiro.category.domain.valueobject;

import static org.assertj.core.api.Assertions.assertThat;

import org.junit.jupiter.api.Test;

class CategoryDescriptionTest {

    @Test
    void validDescriptionReturnsValue() {
        var result = CategoryDescription.create("Alimentação");

        assertThat(result.isValid()).isTrue();
        assertThat(result.value().value()).isEqualTo("Alimentação");
    }

    @Test
    void descriptionWithSpacesIsTrimmed() {
        var result = CategoryDescription.create("  Alimentação  ");

        assertThat(result.value().value()).isEqualTo("Alimentação");
    }

    @Test
    void nullDescriptionIsInvalid() {
        var result = CategoryDescription.create(null);

        assertThat(result.isInvalid()).isTrue();
        assertThat(result.errors()).containsExactly("Descrição da categoria é obrigatória");
    }

    @Test
    void blankDescriptionIsInvalid() {
        var result = CategoryDescription.create("   ");

        assertThat(result.isInvalid()).isTrue();
        assertThat(result.errors()).containsExactly("Descrição da categoria é obrigatória");
    }

    @Test
    void shortDescriptionIsInvalid() {
        var result = CategoryDescription.create("Ab");

        assertThat(result.isInvalid()).isTrue();
        assertThat(result.errors()).containsExactly("Descrição da categoria deve ter entre 3 e 100 caracteres");
    }

    @Test
    void longDescriptionIsInvalid() {
        var result = CategoryDescription.create("A".repeat(101));

        assertThat(result.isInvalid()).isTrue();
        assertThat(result.errors()).containsExactly("Descrição da categoria deve ter entre 3 e 100 caracteres");
    }
}
