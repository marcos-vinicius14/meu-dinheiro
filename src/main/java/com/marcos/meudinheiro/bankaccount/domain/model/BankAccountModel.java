package com.marcos.meudinheiro.bankaccount.domain.model;

import com.marcos.meudinheiro.bankaccount.domain.enums.BankAccountType;
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

    protected BankAccountModel() {}

    private BankAccountModel(
            String name,
            BankAccountType bankAccountType,
            Money initialBalance,
            UserModel user
    ) {
        this.name = name;
        this.bankAccountType = bankAccountType;
        this.initialBalance = initialBalance;
        this.user = user;
    }

    public static BankAccountModel create(
            String name,
            BankAccountType bankAccountType,
            Money initialBalance,
            UserModel user
    ) {
        return new BankAccountModel(
                name,
                bankAccountType,
                initialBalance,
                user
        );
    }

    public void updateName(String name) {
        this.name = name;
    }

    public void updateInitialBalance(Money initialBalance) {
        this.initialBalance = Objects.requireNonNull(initialBalance);
    }

    public boolean belongsTo(UUID userId) {
        return user != null && user.getId().equals(userId);
    }

    public UUID getId() {
        return id;
    }

    public String getName() {
        return name;
    }

    public Money getInitialBalance() {
        return initialBalance;
    }

    public BankAccountType getBankAccountType() {
        return bankAccountType;
    }

    public UserModel getUser() {
        return user;
    }
}
