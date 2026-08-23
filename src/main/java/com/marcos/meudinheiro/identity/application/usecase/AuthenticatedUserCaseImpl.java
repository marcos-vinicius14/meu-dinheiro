package com.marcos.meudinheiro.identity.application.usecase;

import com.marcos.meudinheiro.identity.application.contract.AuthenticateUserUseCase;
import com.marcos.meudinheiro.identity.application.contract.dto.AuthenticationInput;
import com.marcos.meudinheiro.identity.application.contract.dto.AuthenticationOutput;
import com.marcos.meudinheiro.identity.infraestructure.security.JwtTokenService;
import org.springframework.security.authentication.AuthenticationManager;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;

public class AuthenticatedUserCaseImpl implements AuthenticateUserUseCase {
    private final AuthenticationManager authenticationManager;
    private final JwtTokenService jwtTokenService;

    public AuthenticatedUserCaseImpl(AuthenticationManager authenticationManager, JwtTokenService jwtTokenService) {
        this.authenticationManager = authenticationManager;
        this.jwtTokenService = jwtTokenService;
    }

    @Override
    public AuthenticationOutput execute(AuthenticationInput authenticationInput) {
        var authentication =  authenticationManager.authenticate(
                UsernamePasswordAuthenticationToken
                        .unauthenticated(
                                authenticationInput.email(),
                                authenticationInput.password()
                        )
        );

        var token = jwtTokenService.generate(authentication);

        return  AuthenticationOutput.bearer(
                token.value(),
                token.expiresIn()
        );
    }
}
