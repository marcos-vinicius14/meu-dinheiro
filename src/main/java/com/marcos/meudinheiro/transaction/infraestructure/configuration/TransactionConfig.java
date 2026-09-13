package com.marcos.meudinheiro.transaction.infraestructure.configuration;

import com.marcos.meudinheiro.transaction.application.contract.TransactionUserResolver;
import com.marcos.meudinheiro.user.domain.model.UserModel;
import jakarta.persistence.EntityManager;
import jakarta.persistence.PersistenceContext;
import org.springframework.stereotype.Component;

import java.util.UUID;

@Component
public class TransactionConfig implements TransactionUserResolver {

    @PersistenceContext
    private EntityManager entityManager;

    @Override
    public UserModel resolve(UUID userId) {
        return entityManager.getReference(UserModel.class, userId);
    }
}
