package com.marcos.meudinheiro.bankaccount.domain.valueobject;

import static org.assertj.core.api.Assertions.assertThat;

import org.junit.jupiter.api.Test;

class BankAccountNameTest {

    @Test
    void validNameReturnsValue() {
        var result = BankAccountName.create("Minha Conta");

        assertThat(result.isValid()).isTrue();
        assertThat(result.value().value()).isEqualTo("Minha Conta");
    }

    @Test
    void nameWithSpacesIsTrimmed() {
        var result = BankAccountName.create("  Minha Conta  ");

        assertThat(result.value().value()).isEqualTo("Minha Conta");
    }

    @Test
    void nullNameIsInvalid() {
        var result = BankAccountName.create(null);

        assertThat(result.isInvalid()).isTrue();
        assertThat(result.errors()).containsExactly("Nome da conta é obrigatório");
    }

    @Test
    void blankNameIsInvalid() {
        var result = BankAccountName.create("   ");

        assertThat(result.isInvalid()).isTrue();
        assertThat(result.errors()).containsExactly("Nome da conta é obrigatório");
    }

    @Test
    void shortNameIsInvalid() {
        var result = BankAccountName.create("AB");

        assertThat(result.isInvalid()).isTrue();
        assertThat(result.errors()).containsExactly("Nome da conta deve ter entre 3 e 100 caracteres");
    }

    @Test
    void longNameIsInvalid() {
        var result = BankAccountName.create("A".repeat(101));

        assertThat(result.isInvalid()).isTrue();
        assertThat(result.errors()).containsExactly("Nome da conta deve ter entre 3 e 100 caracteres");
    }
}
