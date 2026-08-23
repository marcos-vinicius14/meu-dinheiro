package com.marcos.meudinheiro.user.infraesctructure.web.controller;

import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.user.infraesctructure.web.dto.ApiResponse;
import com.marcos.meudinheiro.user.infraesctructure.web.dto.CreateUserRequest;
import com.marcos.meudinheiro.user.application.usecase.CreateUserCase;
import jakarta.validation.Valid;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/users")
public class UserController {

    private final CreateUserCase userCase;

    public UserController(CreateUserCase userCase) {
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
