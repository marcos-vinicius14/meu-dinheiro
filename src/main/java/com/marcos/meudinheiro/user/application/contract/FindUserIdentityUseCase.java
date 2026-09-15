package com.marcos.meudinheiro.user.application.contract;

import com.marcos.meudinheiro.user.application.contract.dto.UserIdentityOutput;
import java.util.Optional;
import java.util.UUID;

public interface FindUserIdentityUseCase {
  Optional<UserIdentityOutput> findByEmail(String email);

  Optional<UserIdentityOutput> findById(UUID userId);
}
