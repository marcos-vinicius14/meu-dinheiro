package com.marcos.meudinheiro.user.application.mapper;

import com.marcos.meudinheiro.shared.notification.Notification;
import com.marcos.meudinheiro.user.domain.model.UserModel;
import com.marcos.meudinheiro.user.domain.valueobject.Email;
import com.marcos.meudinheiro.user.infraesctructure.web.dto.CreateUserRequest;

public class UserMapper {
  private UserMapper() {}

  public static UserModel toUserModel(CreateUserRequest request, Notification notification) {

    Email email = notification.collect(Email.create(request.email()));
    return UserModel.create(request.username(), email, request.password());
  }
}
