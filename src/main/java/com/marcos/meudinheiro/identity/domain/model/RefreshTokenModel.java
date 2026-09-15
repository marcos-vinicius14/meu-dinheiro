package com.marcos.meudinheiro.identity.domain.model;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.Table;
import java.time.Instant;
import java.util.UUID;

@Entity
@Table(name = "tb_refresh_tokens")
public class RefreshTokenModel {

  @Id
  @GeneratedValue(strategy = GenerationType.UUID)
  private UUID id;

  @Column(name = "user_id", nullable = false)
  private UUID userId;

  @Column(name = "token_hash", nullable = false, unique = true, length = 64)
  private String tokenHash;

  @Column(name = "expires_at", nullable = false)
  private Instant expiresAt;

  @Column(name = "revoked_at")
  private Instant revokedAt;

  @Column(name = "created_at", nullable = false)
  private Instant createdAt;

  protected RefreshTokenModel() {}

  private RefreshTokenModel(UUID userId, String tokenHash, Instant expiresAt, Instant createdAt) {
    this.userId = userId;
    this.tokenHash = tokenHash;
    this.expiresAt = expiresAt;
    this.createdAt = createdAt;
  }

  public static RefreshTokenModel create(
      UUID userId, String tokenHash, Instant expiresAt, Instant createdAt) {
    return new RefreshTokenModel(userId, tokenHash, expiresAt, createdAt);
  }

  public boolean isExpired(Instant now) {
    return !expiresAt.isAfter(now);
  }

  public boolean isRevoked() {
    return revokedAt != null;
  }

  public boolean isValid(Instant now) {
    return !isRevoked() && !isExpired(now);
  }

  public void revoke(Instant now) {
    this.revokedAt = now;
  }

  public UUID getId() {
    return id;
  }

  public UUID getUserId() {
    return userId;
  }

  public String getTokenHash() {
    return tokenHash;
  }
}
