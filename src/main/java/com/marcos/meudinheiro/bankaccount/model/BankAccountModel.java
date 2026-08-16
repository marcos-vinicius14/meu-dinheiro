package com.marcos.meudinheiro.bankaccount.model;

import com.marcos.meudinheiro.bankaccount.enums.BankAccountType;
import com.marcos.meudinheiro.shared.valueobjects.Money;
import com.marcos.meudinheiro.user.domain.model.UserModel;
import jakarta.persistence.*;
import org.hibernate.annotations.Generated;

import java.util.Objects;
import java.util.UUID;

@Entity
@Table(name = "tb_bank_accounts")
public class BankAccountModel {
    @Id
    @Generated
    @Column(
            name = "id",
            insertable = false,
            updatable = false
    )
    private UUID id;

    @Column(nullable = false)
    private String name;

    @Embedded
    @AttributeOverride(
            name = "value",
            column = @Column(
                    name = "initial_balance",
                    nullable = false,
                    precision = 19,
                    scale = 2
            )
    )
    private Money initialBalance;

    @Enumerated(EnumType.STRING)
    @Column(name = "type", nullable = false)
    private BankAccountType bankAccountType;

    @ManyToOne(fetch = FetchType.LAZY, optional = false)
    @JoinColumn(name = "user_id")
    private UserModel user;

    public BankAccountModel() {}

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public Money getInitialBalance() {
        return initialBalance;
    }

    public void setInitialBalance(Money initialBalance) {

        this.initialBalance = Objects.requireNonNull(initialBalance);
    }

    public BankAccountType getBankAccountType() {
        return bankAccountType;
    }

    public void setBankAccountType(BankAccountType bankAccountType) {
        this.bankAccountType = bankAccountType;
    }

    public UserModel getUser() {
        return user;
    }

    public void setUser(UserModel user) {
        this.user = user;
    }
}
