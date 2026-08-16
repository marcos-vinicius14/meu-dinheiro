package com.marcos.meudinheiro.user.infraesctructure.controller;

import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.user.infraesctructure.controller.dto.ApiResponse;
import com.marcos.meudinheiro.user.infraesctructure.controller.dto.request.CreateUserRequest;
import com.marcos.meudinheiro.user.application.usecase.UserCase;
import jakarta.validation.Valid;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/users")
public class UserController {

    private final UserCase userCase;

    public UserController(UserCase userCase) {
        this.userCase = userCase;
    }

    @PostMapping
    public ResponseEntity<ApiResponse> create(
            @Valid @RequestBody CreateUserRequest request
    ) {

        OperationResult result = userCase.execute(request);

        if (result.isFailure()) {
            return ResponseEntity
                    .badRequest()
                    .body(new ApiResponse(result.errors()));
        }

        return ResponseEntity.noContent().build();
    }
}
