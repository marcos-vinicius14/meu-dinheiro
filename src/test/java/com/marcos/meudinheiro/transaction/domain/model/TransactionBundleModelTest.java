package com.marcos.meudinheiro.transaction.domain.model;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import com.marcos.meudinheiro.shared.valueobjects.Money;
import com.marcos.meudinheiro.transaction.domain.enums.TransactionStatus;
import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.UUID;
import org.junit.jupiter.api.Test;

class TransactionBundleModelTest {

  private static final LocalDate FIRST_DUE = LocalDate.of(2026, 9, 15);

  @Test
  void createBundleStoresValues() {
    var bundle = TransactionBundleModel.create("Notebook", money(1000), 10, FIRST_DUE, null);

    assertThat(bundle.getDescription()).isEqualTo("Notebook");
    assertThat(bundle.getTotalAmount().getValue()).isEqualByComparingTo(new BigDecimal("1000.00"));
    assertThat(bundle.getTotalInstallments()).isEqualTo(10);
    assertThat(bundle.getFirstDueDate()).isEqualTo(FIRST_DUE);
  }

  @Test
  void createBundleRejectsNonPositiveInstallments() {
    assertThatThrownBy(() -> TransactionBundleModel.create("X", money(100), 0, FIRST_DUE, null))
        .isInstanceOf(IllegalArgumentException.class);
  }

  @Test
  void createBundleRejectsNonPositiveAmount() {
    assertThatThrownBy(() -> TransactionBundleModel.create("X", money(0), 3, FIRST_DUE, null))
        .isInstanceOf(IllegalArgumentException.class);
  }

  @Test
  void generateInstallmentsCreatesNCommittedTransactions() {
    var bundle = TransactionBundleModel.create("Notebook", money(1000), 10, FIRST_DUE, null);

    var bundleId = UUID.randomUUID();
    var installments = bundle.generateInstallments(bundleId, null, null);

    assertThat(installments).hasSize(10);
    assertThat(installments).allMatch(t -> t.getStatus() == TransactionStatus.COMMITTED);
    assertThat(installments.get(0).getInstallmentNumber()).isEqualTo(1);
    assertThat(installments.get(9).getInstallmentNumber()).isEqualTo(10);
    assertThat(installments).allMatch(t -> t.getBundleId().equals(bundleId));
  }

  @Test
  void generateInstallmentsAllocatesTotalExactly() {
    var bundle = TransactionBundleModel.create("Compra com resto", money(100), 3, FIRST_DUE, null);

    var installments = bundle.generateInstallments(UUID.randomUUID(), null, null);

    var total =
        installments.stream()
            .map(t -> t.getAmount().getValue())
            .reduce(BigDecimal.ZERO, BigDecimal::add);

    assertThat(total).isEqualByComparingTo(new BigDecimal("100.00"));
  }

  @Test
  void generateInstallmentsUsesMonthlyDueDates() {
    var bundle = TransactionBundleModel.create("Parcelado", money(300), 3, FIRST_DUE, null);

    var installments = bundle.generateInstallments(UUID.randomUUID(), null, null);

    assertThat(installments.get(0).getDueDate()).isEqualTo(LocalDate.of(2026, 9, 15));
    assertThat(installments.get(1).getDueDate()).isEqualTo(LocalDate.of(2026, 10, 15));
    assertThat(installments.get(2).getDueDate()).isEqualTo(LocalDate.of(2026, 11, 15));
  }

  @Test
  void generateInstallmentsClampsDayToEndOfMonth() {
    var bundle =
        TransactionBundleModel.create(
            "Parcelado fim de mês", money(300), 3, LocalDate.of(2026, 1, 31), null);

    var installments = bundle.generateInstallments(UUID.randomUUID(), null, null);

    assertThat(installments.get(0).getDueDate()).isEqualTo(LocalDate.of(2026, 1, 31));
    assertThat(installments.get(1).getDueDate()).isEqualTo(LocalDate.of(2026, 2, 28));
    assertThat(installments.get(2).getDueDate()).isEqualTo(LocalDate.of(2026, 3, 31));
  }

  @Test
  void generateInstallmentsCarriesUserAndCategory() {
    var bundle = TransactionBundleModel.create("Parcelado", money(300), 2, FIRST_DUE, null);

    var installments = bundle.generateInstallments(UUID.randomUUID(), null, null);

    assertThat(installments).hasSize(2);
    assertThat(installments.get(0).getDescription()).isEqualTo("Parcelado");
  }

  private static Money money(int value) {
    return new Money(new BigDecimal(value));
  }
}
