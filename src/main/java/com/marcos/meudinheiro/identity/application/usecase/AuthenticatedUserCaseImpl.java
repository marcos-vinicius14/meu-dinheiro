package com.marcos.meudinheiro.identity.application.usecase;

import com.marcos.meudinheiro.identity.application.contract.AuthenticateUserUseCase;
import com.marcos.meudinheiro.identity.application.contract.LoginAttemptLimiter;
import com.marcos.meudinheiro.identity.application.contract.dto.AuthenticationInput;
import com.marcos.meudinheiro.identity.application.contract.dto.AuthenticationOutput;
import com.marcos.meudinheiro.identity.domain.model.RefreshTokenModel;
import com.marcos.meudinheiro.identity.infraestructure.repository.RefreshTokenRepository;
import com.marcos.meudinheiro.identity.infraestructure.security.authentication.AuthenticadedUser;
import com.marcos.meudinheiro.identity.infraestructure.security.authentication.JwtSubject;
import com.marcos.meudinheiro.identity.infraestructure.security.token.JwtTokenService;
import com.marcos.meudinheiro.identity.infraestructure.security.token.RefreshTokenGenerator;
import com.marcos.meudinheiro.shared.notification.OperationResult;

import java.time.Clock;
import java.time.Duration;
import java.util.Locale;

import org.springframework.security.authentication.AuthenticationManager;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.AuthenticationException;
import org.springframework.stereotype.Service;

@Service
public class AuthenticatedUserCaseImpl implements AuthenticateUserUseCase {
    private static final String INVALID_CREDENTIALS = "Credenciais inválidas";

    private final AuthenticationManager authenticationManager;
    private final JwtTokenService jwtTokenService;
    private final RefreshTokenGenerator refreshTokenGenerator;
    private final RefreshTokenRepository refreshTokenRepository;
    private final LoginAttemptLimiter loginAttemptLimiter;
    private final Clock clock;

    private static final Duration REFRESH_TOKEN_TTL = Duration.ofDays(15);

    public AuthenticatedUserCaseImpl(
        AuthenticationManager authenticationManager, 
        JwtTokenService jwtTokenService,
        RefreshTokenGenerator refreshTokenGenerator,
        RefreshTokenRepository refreshTokenRepository,
        LoginAttemptLimiter loginAttemptLimiter,
        Clock clock
    ) {
        this.authenticationManager = authenticationManager;
        this.jwtTokenService = jwtTokenService;
        this.refreshTokenGenerator = refreshTokenGenerator;
        this.refreshTokenRepository = refreshTokenRepository;
        this.loginAttemptLimiter = loginAttemptLimiter;
        this.clock = clock;
    }

    @Override
    public OperationResult<AuthenticationOutput> execute(AuthenticationInput authenticationInput) {
        var email = authenticationInput.email()
                .trim()
                .toLowerCase(Locale.ROOT);

        if (loginAttemptLimiter.isBlocked(email)) {
            return OperationResult.failure(INVALID_CREDENTIALS);
        }

        try {
            var authentication = authenticationManager.authenticate(
                    UsernamePasswordAuthenticationToken
                            .unauthenticated(
                                    email,
                                    authenticationInput.password()
                            )
            );

            loginAttemptLimiter.reset(email);

            return OperationResult.success(
                    issueSession((AuthenticadedUser) authentication.getPrincipal())
            );
        } catch (AuthenticationException exception) {
            loginAttemptLimiter.recordFailure(email);

            return OperationResult.failure(INVALID_CREDENTIALS);
        }
    }

    private AuthenticationOutput issueSession(AuthenticadedUser principal) {
        var accessToken = jwtTokenService.generate(
            new JwtSubject(
                principal.id(),
                principal.email()
            )
        );

        var generatedRefreshToken = refreshTokenGenerator.generate();

        var now = clock.instant(); 

        var expiresAt = now.plus(REFRESH_TOKEN_TTL);

        var refreshToken = RefreshTokenModel.create(
            principal.id(), 
            generatedRefreshToken.hash(), 
            expiresAt, 
            now
        );

        refreshTokenRepository.save(refreshToken);

        return AuthenticationOutput.bearer(
                accessToken.value(),
                accessToken.expiresIn(),
                generatedRefreshToken.value(),
                REFRESH_TOKEN_TTL.toSeconds()
        );
    }
}
