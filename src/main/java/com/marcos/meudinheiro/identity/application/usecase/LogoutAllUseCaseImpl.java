package com.marcos.meudinheiro.identity.application.usecase;

import java.time.Clock;

import org.springframework.stereotype.Service;

import com.marcos.meudinheiro.identity.application.contract.CurrentIdentity;
import com.marcos.meudinheiro.identity.application.contract.LogoutAllUseCase;
import com.marcos.meudinheiro.identity.infraestructure.repository.RefreshTokenRepository;
import com.marcos.meudinheiro.shared.notification.OperationResult;

import jakarta.transaction.Transactional;

@Service
public class LogoutAllUseCaseImpl implements LogoutAllUseCase {

    private final CurrentIdentity currentIdentity;
    private final RefreshTokenRepository refreshTokenRepository;
    private final Clock clock;

    LogoutAllUseCaseImpl(
        CurrentIdentity currentIdentity,
        RefreshTokenRepository refreshTokenRepository,
        Clock clock
    ) {
        this.currentIdentity = currentIdentity;
        this.refreshTokenRepository = refreshTokenRepository;
        this.clock = clock;
    }

    @Transactional
    @Override
    public OperationResult<Void> execute() {
        var userId = currentIdentity.findCurrentAuthenticadedUser();

        refreshTokenRepository.revokeAllByUserId(
            userId,
            clock.instant()
        );

        return OperationResult.success();
    }
}
