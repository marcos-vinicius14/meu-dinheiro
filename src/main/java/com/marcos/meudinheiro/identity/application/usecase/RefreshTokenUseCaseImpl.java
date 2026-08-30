package com.marcos.meudinheiro.identity.application.usecase;

import java.time.Clock;
import java.time.Duration;
import java.time.Instant;
import java.util.Optional;
import java.util.UUID;

import org.springframework.stereotype.Service;

import com.marcos.meudinheiro.identity.application.contract.RefreshTokenUseCase;
import com.marcos.meudinheiro.identity.application.contract.dto.AuthenticationOutput;
import com.marcos.meudinheiro.identity.domain.model.RefreshTokenModel;
import com.marcos.meudinheiro.identity.infraestructure.repository.RefreshTokenRepository;
import com.marcos.meudinheiro.identity.infraestructure.security.authentication.JwtSubject;
import com.marcos.meudinheiro.identity.infraestructure.security.token.JwtTokenService;
import com.marcos.meudinheiro.identity.infraestructure.security.token.RefreshTokenGenerator;
import com.marcos.meudinheiro.identity.infraestructure.security.token.RefreshTokenHash;
import com.marcos.meudinheiro.shared.notification.Notification;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.shared.notification.ValidationResult;
import com.marcos.meudinheiro.user.application.contract.FindUserIdentityUseCase;
import com.marcos.meudinheiro.user.application.contract.dto.UserIdentityOutput;

import jakarta.transaction.Transactional;

@Service
public class RefreshTokenUseCaseImpl implements RefreshTokenUseCase {
    private static final Duration REFRESH_TOKEN_TTL = Duration.ofDays(15);
    private static final String INVALID_REFRESH_TOKEN = "Refresh token inválido, expirado ou revogado";


    private final RefreshTokenRepository refreshTokenRepository;
    private final RefreshTokenGenerator refreshTokenGenerator;
    private final JwtTokenService jwtTokenService;
    private final FindUserIdentityUseCase findUserIdentity;
    private final Clock clock;

    RefreshTokenUseCaseImpl (
    RefreshTokenRepository refreshTokenRepository,
    RefreshTokenGenerator refreshTokenGenerator,
    JwtTokenService jwtTokenService,
    FindUserIdentityUseCase findUserIdentity,
    Clock clock
) {
    this.refreshTokenRepository = refreshTokenRepository;
    this.refreshTokenGenerator = refreshTokenGenerator;
    this.jwtTokenService = jwtTokenService;
    this.findUserIdentity = findUserIdentity;
    this.clock = clock;
}

@Transactional
@Override
public OperationResult<AuthenticationOutput> execute(String rawRefreshToken) {
    var notification = new Notification();

    var now = clock.instant();

    var refreshToken = notification.collect(
        resolveRefreshToken(rawRefreshToken, now)
    );

    if (notification.hasErrors()) {
        return OperationResult.failure(notification.errors());
    }

    refreshToken.revoke(now);

    var identity = notification.collect(
        resolveIdentity(refreshToken.getUserId())
    );

    if (notification.hasErrors()) {
        return OperationResult.failure(notification.errors());
    }

    var accessToken =
        jwtTokenService.generate(
            new JwtSubject(
                identity.id(),
                identity.email()
            )
        );

    var generatedRefreshToken =
        refreshTokenGenerator.generate();

    var newRefreshToken =
        RefreshTokenModel.create(
            identity.id(),
            generatedRefreshToken.hash(),
            now.plus(REFRESH_TOKEN_TTL),
            now
        );

    refreshTokenRepository.save(newRefreshToken);

    return OperationResult.success(
        AuthenticationOutput.bearer(
            accessToken.value(),
            accessToken.expiresIn(),
            generatedRefreshToken.value(),
            REFRESH_TOKEN_TTL.toSeconds()
        )
    );
}

    private ValidationResult<RefreshTokenModel> resolveRefreshToken(
        String rawRefreshToken,
        Instant now
    ) {
        if (rawRefreshToken == null || rawRefreshToken.isBlank()) {
            return ValidationResult.invalid(INVALID_REFRESH_TOKEN);
        }

        var refreshToken = refreshTokenRepository
            .findByTokenHash(
                RefreshTokenHash.sha256(rawRefreshToken)
            )
            .orElse(null);

     
        return Optional.ofNullable(refreshToken)
                .map(token -> validateRefreshToken(token, now))
                .orElseGet(() -> ValidationResult.invalid(INVALID_REFRESH_TOKEN));
    }

    private void revokeAllUserTokens(UUID userId, Instant now) {
        refreshTokenRepository.revokeAllByUserId(userId, now);
    }

    private ValidationResult<UserIdentityOutput> resolveIdentity(UUID userId) {
        return findUserIdentity
            .findById(userId)
            .<ValidationResult<UserIdentityOutput>>map(ValidationResult::valid)
            .orElseGet(() ->
                ValidationResult.invalid(INVALID_REFRESH_TOKEN)
            );
    }

    private ValidationResult<RefreshTokenModel> validateRefreshToken(
        RefreshTokenModel refreshToken,
        Instant now
    ) {

        if (refreshToken.isRevoked()) {
            revokeAllUserTokens(refreshToken.getUserId(), now);
            return ValidationResult.invalid(INVALID_REFRESH_TOKEN);
        }

        if (refreshToken.isExpired(now)) {
            return ValidationResult.invalid(INVALID_REFRESH_TOKEN);
        }

        return ValidationResult.valid(refreshToken);
    }
}
