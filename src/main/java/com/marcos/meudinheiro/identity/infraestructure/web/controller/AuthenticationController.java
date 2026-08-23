package com.marcos.meudinheiro.identity.infraestructure.web.controller;

import com.marcos.meudinheiro.identity.application.contract.AuthenticateUserUseCase;
import com.marcos.meudinheiro.identity.application.contract.dto.AuthenticationInput;
import com.marcos.meudinheiro.identity.application.contract.dto.AuthenticationOutput;
import com.marcos.meudinheiro.identity.infraestructure.web.dto.LoginRequest;
import jakarta.servlet.http.HttpServletResponse;
import jakarta.validation.Valid;
import org.springframework.http.HttpHeaders;
import org.springframework.http.ResponseCookie;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/auth")
final class AuthenticationController {
    private final AuthenticateUserUseCase useCase;

    AuthenticationController(AuthenticateUserUseCase useCase) {
        this.useCase = useCase;
    }

    @PostMapping("/login")
    ResponseEntity<Void> login(
            @Valid  @RequestBody LoginRequest request,
            HttpServletResponse response
            ) {

        //TODO: Mover para um mapper
        var input = new AuthenticationInput(
                request.email(),
                request.password()
        );

        var authentication = useCase.execute(input);

        var cookie = ResponseCookie.from(
                "access-token",
                authentication.acessToken()
        )
                .httpOnly(true)
                .secure(true)
                .sameSite("strict")
                .path("/")
                .maxAge(authentication.expiresIn())
                .build();

        response.addHeader(
                HttpHeaders.SET_COOKIE,
                cookie.toString()
        );

        return ResponseEntity
                .noContent().
                build();

    }
}
