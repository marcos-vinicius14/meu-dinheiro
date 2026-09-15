package com.marcos.meudinheiro.transaction.domain.model;

import com.marcos.meudinheiro.shared.valueobjects.Money;
import com.marcos.meudinheiro.transaction.domain.enums.HealthStatus;
import jakarta.persistence.*;
import java.time.LocalDate;
import java.util.UUID;
import org.hibernate.annotations.Generated;

@Entity
@Table(name = "tb_check_in_snapshots")
public class CheckInSnapshotModel {

  @Id
  @Generated
  @Column(name = "id", insertable = false, updatable = false)
  private UUID id;

  @ManyToOne(fetch = FetchType.LAZY, optional = false)
  @JoinColumn(name = "user_id", nullable = false)
  private com.marcos.meudinheiro.user.domain.model.UserModel user;

  @Column(name = "check_in_date", nullable = false)
  private LocalDate checkInDate;

  @Embedded
  @AttributeOverride(
      name = "value",
      column = @Column(name = "s2s_calculated", nullable = false, precision = 19, scale = 2))
  private Money s2sCalculated;

  @Embedded
  @AttributeOverride(
      name = "value",
      column = @Column(name = "spent_today", nullable = false, precision = 19, scale = 2))
  private Money spentToday;

  @Embedded
  @AttributeOverride(
      name = "value",
      column =
          @Column(name = "delta_from_safe_to_spend", nullable = false, precision = 19, scale = 2))
  private Money deltaFromSafeToSpend;

  @Enumerated(EnumType.STRING)
  @Column(name = "health_status", nullable = false)
  private HealthStatus healthStatus;

  protected CheckInSnapshotModel() {}

  public CheckInSnapshotModel(
      com.marcos.meudinheiro.user.domain.model.UserModel user,
      LocalDate checkInDate,
      Money s2sCalculated,
      Money spentToday,
      Money deltaFromSafeToSpend,
      HealthStatus healthStatus) {
    this.user = user;
    this.checkInDate = checkInDate;
    this.s2sCalculated = s2sCalculated;
    this.spentToday = spentToday;
    this.deltaFromSafeToSpend = deltaFromSafeToSpend;
    this.healthStatus = healthStatus;
  }

  public UUID getId() {
    return id;
  }

  public LocalDate getCheckInDate() {
    return checkInDate;
  }

  public Money getS2sCalculated() {
    return s2sCalculated;
  }

  public Money getSpentToday() {
    return spentToday;
  }

  public Money getDeltaFromSafeToSpend() {
    return deltaFromSafeToSpend;
  }

  public HealthStatus getHealthStatus() {
    return healthStatus;
  }

  /** Atualiza o snapshot do dia (idempotência: re-check-in sobrescreve o mesmo registro). */
  public void update(
      Money s2sCalculated,
      Money spentToday,
      Money deltaFromSafeToSpend,
      HealthStatus healthStatus) {
    this.s2sCalculated = s2sCalculated;
    this.spentToday = spentToday;
    this.deltaFromSafeToSpend = deltaFromSafeToSpend;
    this.healthStatus = healthStatus;
  }
}
