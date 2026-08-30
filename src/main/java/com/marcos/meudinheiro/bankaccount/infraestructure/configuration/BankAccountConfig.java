package com.marcos.meudinheiro.bankaccount.infraestructure.configuration;

import com.marcos.meudinheiro.bankaccount.application.contract.BankAccountUserResolver;
import com.marcos.meudinheiro.user.domain.model.UserModel;
import jakarta.persistence.EntityManager;
import jakarta.persistence.PersistenceContext;
import org.springframework.stereotype.Component;

import java.util.UUID;

@Component
public class BankAccountConfig implements BankAccountUserResolver {

    @PersistenceContext
    private EntityManager entityManager;

    @Override
    public UserModel resolve(UUID userId) {
        return entityManager.getReference(UserModel.class, userId);
    }
}
