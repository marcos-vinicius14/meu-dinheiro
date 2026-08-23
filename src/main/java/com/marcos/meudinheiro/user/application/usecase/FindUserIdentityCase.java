package com.marcos.meudinheiro.user.application.usecase;

import com.marcos.meudinheiro.user.application.contract.FindUserIdentityUseCase;
import com.marcos.meudinheiro.user.application.contract.dto.UserIdentityOutput;
import com.marcos.meudinheiro.user.infraesctructure.repository.UserRepository;
import org.springframework.stereotype.Service;

import java.util.Optional;

@Service
public final class FindUserIdentityCase implements FindUserIdentityUseCase {
    private final UserRepository userRepository;

    public FindUserIdentityCase(UserRepository userRepository) {
        this.userRepository = userRepository;
    }

    @Override
    public Optional<UserIdentityOutput> execute(String email) {
        return userRepository.findByEmail(email)
                .map(user -> new UserIdentityOutput(
                        user.getId(),
                        user.getEmail().value(),
                        user.getPassword()
                ));
    }
}
