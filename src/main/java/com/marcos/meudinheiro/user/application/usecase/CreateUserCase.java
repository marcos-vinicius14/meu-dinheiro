package com.marcos.meudinheiro.user.application.usecase;

import com.marcos.meudinheiro.shared.notification.Notification;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.user.infraesctructure.web.dto.CreateUserRequest;
import com.marcos.meudinheiro.user.application.mapper.UserMapper;
import com.marcos.meudinheiro.user.domain.model.UserModel;
import com.marcos.meudinheiro.user.infraesctructure.repository.UserRepository;

import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;

@Service
public final class CreateUserCase {
    private final UserRepository repository;
    private final PasswordEncoder passwordEncoder;

    public CreateUserCase(UserRepository repository, PasswordEncoder passwordEncoder) {
        this.repository = repository;
        this.passwordEncoder = passwordEncoder;
    }

    public OperationResult execute(CreateUserRequest request) {
        Notification notification = new Notification();

        UserModel user = UserMapper.toUserModel(request, notification);

        if (notification.hasErrors()) {
            return OperationResult.failure(notification.errors());
        }

        user.setPassword(passwordEncoder.encode(request.password()));
        repository.save(user);

        return OperationResult.success();
    }
}
