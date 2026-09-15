package com.marcos.meudinheiro.user.application.usecase;

import com.marcos.meudinheiro.user.application.contract.FindUserIdentityUseCase;
import com.marcos.meudinheiro.user.application.contract.dto.UserIdentityOutput;
import com.marcos.meudinheiro.user.infraesctructure.repository.UserRepository;
import java.util.Optional;
import java.util.UUID;
import org.springframework.stereotype.Service;

@Service
public final class FindUserIdentityUseCaseImpl implements FindUserIdentityUseCase {
  private final UserRepository userRepository;

  public FindUserIdentityUseCaseImpl(UserRepository userRepository) {
    this.userRepository = userRepository;
  }

  @Override
  public Optional<UserIdentityOutput> findByEmail(String email) {
    return userRepository
        .findByEmail(email)
        .map(
            user ->
                new UserIdentityOutput(user.getId(), user.getEmail().value(), user.getPassword()));
  }

  @Override
  public Optional<UserIdentityOutput> findById(UUID id) {
    return userRepository
        .findById(id)
        .map(
            user ->
                new UserIdentityOutput(user.getId(), user.getEmail().value(), user.getPassword()));
  }
}
