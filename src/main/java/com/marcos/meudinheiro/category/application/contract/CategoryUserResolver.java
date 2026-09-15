package com.marcos.meudinheiro.category.application.contract;

import com.marcos.meudinheiro.user.domain.model.UserModel;
import java.util.UUID;

public interface CategoryUserResolver {
  UserModel resolve(UUID userId);
}
