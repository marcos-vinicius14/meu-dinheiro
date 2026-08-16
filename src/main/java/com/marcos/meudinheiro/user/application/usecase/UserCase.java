package com.marcos.meudinheiro.user.application.usecase;

import com.marcos.meudinheiro.shared.notification.Notification;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.user.infraesctructure.controller.dto.request.CreateUserRequest;
import com.marcos.meudinheiro.user.application.mapper.UserMapper;
import com.marcos.meudinheiro.user.domain.model.UserModel;
import com.marcos.meudinheiro.user.infraesctructure.repository.UserRepository;
import org.springframework.stereotype.Service;

@Service
public final class UserCase {
    private final UserRepository repository;

    public UserCase(UserRepository repository) {
        this.repository = repository;
    }

    public OperationResult execute(CreateUserRequest request) {
        Notification notification = new Notification();

        UserModel user = UserMapper.toUserModel(request, notification);

        if (notification.hasErrors()) {
            return OperationResult.failure(notification.errors());
        }

        repository.save(user);

        return  OperationResult.success();
    }
}
