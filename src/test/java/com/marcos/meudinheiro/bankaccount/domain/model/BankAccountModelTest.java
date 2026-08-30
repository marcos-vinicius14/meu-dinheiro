package com.marcos.meudinheiro.bankaccount.domain.model;

import static org.assertj.core.api.Assertions.assertThat;

import java.math.BigDecimal;
import java.util.UUID;

import com.marcos.meudinheiro.bankaccount.domain.enums.BankAccountType;
import com.marcos.meudinheiro.shared.valueobjects.Money;
import com.marcos.meudinheiro.user.domain.model.UserModel;
import com.marcos.meudinheiro.user.domain.valueobject.Email;
import org.junit.jupiter.api.Test;

class BankAccountModelTest {

    @Test
    void createBankAccountStoresValues() {
        var user = createUser();
        var account = BankAccountModel.create(
                "Carteira",
                BankAccountType.CASH,
                new Money(BigDecimal.TEN),
                user
        );

        assertThat(account.getName()).isEqualTo("Carteira");
        assertThat(account.getBankAccountType()).isEqualTo(BankAccountType.CASH);
        assertThat(account.getInitialBalance().getValue()).isEqualByComparingTo(BigDecimal.TEN);
        assertThat(account.getUser()).isEqualTo(user);
    }

    @Test
    void belongsToReturnsTrueForSameUser() {
        var user = createUser();
        var account = BankAccountModel.create(
                "Conta",
                BankAccountType.CHECKING,
                new Money(BigDecimal.ZERO),
                user
        );

        assertThat(account.belongsTo(user.getId())).isTrue();
    }

    @Test
    void belongsToReturnsFalseForDifferentUser() {
        var owner = createUser();
        var other = createUser();
        var account = BankAccountModel.create(
                "Conta",
                BankAccountType.CHECKING,
                new Money(BigDecimal.ZERO),
                owner
        );

        assertThat(account.belongsTo(other.getId())).isFalse();
    }

    @Test
    void updateNameChangesName() {
        var account = BankAccountModel.create(
                "Antigo",
                BankAccountType.CHECKING,
                new Money(BigDecimal.ZERO),
                createUser()
        );

        account.updateName("Novo");

        assertThat(account.getName()).isEqualTo("Novo");
    }

    @Test
    void updateInitialBalanceChangesBalance() {
        var account = BankAccountModel.create(
                "Conta",
                BankAccountType.CHECKING,
                new Money(BigDecimal.ZERO),
                createUser()
        );

        account.updateInitialBalance(new Money(new BigDecimal("150.75")));

        assertThat(account.getInitialBalance().getValue()).isEqualByComparingTo(new BigDecimal("150.75"));
    }

    private UserModel createUser() {
        var email = Email.create("user-" + UUID.randomUUID() + "@example.com");
        var user = UserModel.create(
                "Test User",
                email.value(),
                "password123"
        );
        setUserId(user, UUID.randomUUID());
        return user;
    }

    private void setUserId(UserModel user, UUID id) {
        try {
            var field = UserModel.class.getDeclaredField("id");
            field.setAccessible(true);
            field.set(user, id);
        } catch (NoSuchFieldException | IllegalAccessException e) {
            throw new RuntimeException(e);
        }
    }
}
