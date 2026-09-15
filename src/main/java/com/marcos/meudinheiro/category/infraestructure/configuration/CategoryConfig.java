package com.marcos.meudinheiro.category.infraestructure.configuration;

import com.marcos.meudinheiro.category.application.contract.CategoryUserResolver;
import com.marcos.meudinheiro.user.domain.model.UserModel;
import jakarta.persistence.EntityManager;
import jakarta.persistence.PersistenceContext;
import java.util.UUID;
import org.springframework.stereotype.Component;

@Component
public class CategoryConfig implements CategoryUserResolver {

  @PersistenceContext private EntityManager entityManager;

  @Override
  public UserModel resolve(UUID userId) {
    return entityManager.getReference(UserModel.class, userId);
  }
}
