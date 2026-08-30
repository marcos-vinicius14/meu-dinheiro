package com.marcos.meudinheiro.identity.infraestructure.repository;

import java.time.Instant;
import java.util.Optional;
import java.util.UUID;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import com.marcos.meudinheiro.identity.domain.model.RefreshTokenModel;

@Repository
public interface RefreshTokenRepository extends JpaRepository<RefreshTokenModel, UUID>  {
    Optional<RefreshTokenModel> findByTokenHash(String tokenHash);

    @Modifying
    @Query("""
            UPDATE RefreshTokenModel rt
            SET rt.revokedAt = :now
            WHERE rt.userId = :userId
              AND rt.revokedAt IS NULL
            """)
    int revokeAllByUserId(
        @Param("userId") UUID userId,
        @Param("now") Instant now
    );

    @Modifying
    @Query("""
            DELETE FROM RefreshTokenModel rt
            WHERE rt.expiresAt < :cutoff
               OR rt.revokedAt < :cutoff
            """)
    int deleteStale(@Param("cutoff") Instant cutoff);
}
