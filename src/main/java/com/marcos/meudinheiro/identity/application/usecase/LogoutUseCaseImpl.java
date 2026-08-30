package com.marcos.meudinheiro.identity.application.usecase;

import java.time.Clock;

import org.springframework.stereotype.Service;

import com.marcos.meudinheiro.identity.application.contract.LogoutUseCase;
import com.marcos.meudinheiro.identity.infraestructure.repository.RefreshTokenRepository;
import com.marcos.meudinheiro.identity.infraestructure.security.token.RefreshTokenHash;

import jakarta.transaction.Transactional;

@Service
public class LogoutUseCaseImpl implements LogoutUseCase {

    private final RefreshTokenRepository refreshTokenRepository;
    private final Clock clock;

    LogoutUseCaseImpl(
        RefreshTokenRepository refreshTokenRepository,
        Clock clock
    ) {
        this.refreshTokenRepository = refreshTokenRepository;
        this.clock = clock;
    }

    @Transactional
    @Override
    public void execute(String rawRefreshToken) {
        var tokenHash =
            RefreshTokenHash.sha256(rawRefreshToken);

        refreshTokenRepository
            .findByTokenHash(tokenHash)
            .ifPresent(refreshToken ->
                refreshToken.revoke(clock.instant())
            );
    }
}
