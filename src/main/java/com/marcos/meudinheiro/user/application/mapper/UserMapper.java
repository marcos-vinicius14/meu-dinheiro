package com.marcos.meudinheiro.user.application.mapper;

import com.marcos.meudinheiro.shared.notification.Notification;
import com.marcos.meudinheiro.user.domain.valueobject.Email;
import com.marcos.meudinheiro.user.infraesctructure.controller.dto.request.CreateUserRequest;
import com.marcos.meudinheiro.user.domain.model.UserModel;

public class UserMapper {
    private UserMapper() {
    }

    public static UserModel toUserModel(CreateUserRequest request, Notification notification) {

        Email email = notification.collect(
                Email.create(request.email())
        );
        return UserModel.create(
                request.username(),
                email,
                request.password()
        );
    }
}
