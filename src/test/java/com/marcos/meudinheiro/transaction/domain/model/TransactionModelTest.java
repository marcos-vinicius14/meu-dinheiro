package com.marcos.meudinheiro.transaction.domain.model;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import com.marcos.meudinheiro.shared.valueobjects.Money;
import com.marcos.meudinheiro.transaction.domain.enums.TransactionStatus;
import com.marcos.meudinheiro.transaction.domain.enums.TransactionType;
import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.UUID;
import org.junit.jupiter.api.Test;

class TransactionModelTest {

  private static final LocalDate DUE = LocalDate.of(2026, 9, 20);

  @Test
  void createProjectedTransactionStoresValues() {
    var transaction =
        TransactionModel.create(
            "Supermercado", TransactionType.FLEXIBLE_EXPENSE, money(150), DUE, null, null, null);

    assertThat(transaction.getDescription()).isEqualTo("Supermercado");
    assertThat(transaction.getType()).isEqualTo(TransactionType.FLEXIBLE_EXPENSE);
    assertThat(transaction.getStatus()).isEqualTo(TransactionStatus.PROJECTED);
    assertThat(transaction.getAmount().getValue()).isEqualByComparingTo(new BigDecimal("150.00"));
    assertThat(transaction.getDueDate()).isEqualTo(DUE);
    assertThat(transaction.getPaymentDate()).isNull();
    assertThat(transaction.getBundleId()).isNull();
  }

  @Test
  void createWithNullDescriptionStoresNull() {
    var transaction =
        TransactionModel.create(
            null, TransactionType.FIXED_EXPENSE, money(100), DUE, null, null, null);

    assertThat(transaction.getDescription()).isNull();
  }

  @Test
  void createRejectsNegativeAmount() {
    assertThatThrownBy(
            () ->
                TransactionModel.create(
                    "Negativa", TransactionType.FLEXIBLE_EXPENSE, money(-1), DUE, null, null, null))
        .isInstanceOf(IllegalArgumentException.class);
  }

  @Test
  void createRejectsZeroAmount() {
    assertThatThrownBy(
            () ->
                TransactionModel.create(
                    "Zero", TransactionType.FLEXIBLE_EXPENSE, money(0), DUE, null, null, null))
        .isInstanceOf(IllegalArgumentException.class);
  }

  @Test
  void createRejectsInstallmentTypeOutsideBundle() {
    assertThatThrownBy(
            () ->
                TransactionModel.create(
                    "Avulsa",
                    TransactionType.INSTALLMENT_EXPENSE,
                    money(100),
                    DUE,
                    null,
                    null,
                    null))
        .isInstanceOf(IllegalArgumentException.class);
  }

  @Test
  void createInstallmentInsideBundleIsAllowed() {
    var bundleId = UUID.randomUUID();
    var transaction =
        TransactionModel.createInstallment(
            "Notebook", money(300), DUE, bundleId, 1, 10, null, null, null);

    assertThat(transaction.getType()).isEqualTo(TransactionType.INSTALLMENT_EXPENSE);
    assertThat(transaction.getStatus()).isEqualTo(TransactionStatus.COMMITTED);
    assertThat(transaction.getBundleId()).isEqualTo(bundleId);
    assertThat(transaction.getInstallmentNumber()).isEqualTo(1);
    assertThat(transaction.getTotalInstallments()).isEqualTo(10);
  }

  @Test
  void createInstallmentRejectsInvalidInstallmentRange() {
    var bundleId = UUID.randomUUID();

    assertThatThrownBy(
            () ->
                TransactionModel.createInstallment(
                    "Notebook", money(300), DUE, bundleId, 0, 10, null, null, null))
        .isInstanceOf(IllegalArgumentException.class);

    assertThatThrownBy(
            () ->
                TransactionModel.createInstallment(
                    "Notebook", money(300), DUE, bundleId, 11, 10, null, null, null))
        .isInstanceOf(IllegalArgumentException.class);
  }

  @Test
  void confirmSetsStatusAndPaymentDate() {
    var transaction =
        TransactionModel.create(
            "Conta", TransactionType.FIXED_EXPENSE, money(200), DUE, null, null, null);

    var paidOn = LocalDate.of(2026, 9, 18);
    transaction.confirm(paidOn);

    assertThat(transaction.getStatus()).isEqualTo(TransactionStatus.CONFIRMED);
    assertThat(transaction.getPaymentDate()).isEqualTo(paidOn);
  }

  @Test
  void confirmWithPaymentDateBeforeDueDateIsAllowedForEarlyPayment() {
    var transaction =
        TransactionModel.create(
            "Conta", TransactionType.FIXED_EXPENSE, money(200), DUE, null, null, null);

    var paidOn = DUE.minusDays(2);
    transaction.confirm(paidOn);

    assertThat(transaction.getStatus()).isEqualTo(TransactionStatus.CONFIRMED);
    assertThat(transaction.getPaymentDate()).isEqualTo(paidOn);
  }

  @Test
  void confirmAlreadyConfirmedIsIdempotent() {
    var transaction =
        TransactionModel.create(
            "Conta", TransactionType.FIXED_EXPENSE, money(200), DUE, null, null, null);

    transaction.confirm(DUE);
    transaction.confirm(DUE);

    assertThat(transaction.getStatus()).isEqualTo(TransactionStatus.CONFIRMED);
    assertThat(transaction.getPaymentDate()).isEqualTo(DUE);
  }

  @Test
  void confirmCanceledTransactionIsRejected() {
    var transaction =
        TransactionModel.create(
            "Conta", TransactionType.FIXED_EXPENSE, money(200), DUE, null, null, null);

    transaction.cancel();

    assertThatThrownBy(() -> transaction.confirm(DUE)).isInstanceOf(IllegalStateException.class);
  }

  @Test
  void cancelSetsStatusToCanceled() {
    var transaction =
        TransactionModel.create(
            "Conta", TransactionType.FIXED_EXPENSE, money(200), DUE, null, null, null);

    transaction.cancel();

    assertThat(transaction.getStatus()).isEqualTo(TransactionStatus.CANCELED);
  }

  @Test
  void cancelConfirmedTransactionIsRejected() {
    var transaction =
        TransactionModel.create(
            "Conta", TransactionType.FIXED_EXPENSE, money(200), DUE, null, null, null);

    transaction.confirm(DUE);

    assertThatThrownBy(transaction::cancel).isInstanceOf(IllegalStateException.class);
  }

  @Test
  void belongsToMatchesUserId() {
    var transaction =
        TransactionModel.create(
            "Conta", TransactionType.FIXED_EXPENSE, money(200), DUE, createUser(), null, null);

    assertThat(transaction.belongsTo(transaction.getUser().getId())).isTrue();
    assertThat(transaction.belongsTo(UUID.randomUUID())).isFalse();
  }

  private com.marcos.meudinheiro.user.domain.model.UserModel createUser() {
    var email =
        com.marcos.meudinheiro.user.domain.valueobject.Email.create(
            "user-" + UUID.randomUUID() + "@example.com");
    var user =
        com.marcos.meudinheiro.user.domain.model.UserModel.create(
            "Test User", email.value(), "password123");
    try {
      var field = com.marcos.meudinheiro.user.domain.model.UserModel.class.getDeclaredField("id");
      field.setAccessible(true);
      field.set(user, UUID.randomUUID());
    } catch (NoSuchFieldException | IllegalAccessException e) {
      throw new RuntimeException(e);
    }
    return user;
  }

  private static Money money(int value) {
    return new Money(new BigDecimal(value));
  }
}
