package com.marcos.meudinheiro.shared.valueobjects;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import java.math.BigDecimal;
import org.junit.jupiter.api.Test;

class MoneyTest {

  @Test
  void addSumsValues() {
    var result = new Money(new BigDecimal("10.50")).add(new Money(new BigDecimal("2.25")));

    assertThat(result.getValue()).isEqualByComparingTo(new BigDecimal("12.75"));
  }

  @Test
  void subtractReturnsDifference() {
    var result = new Money(new BigDecimal("10.00")).subtract(new Money(new BigDecimal("3.50")));

    assertThat(result.getValue()).isEqualByComparingTo(new BigDecimal("6.50"));
  }

  @Test
  void multiplyScalesValue() {
    var result = new Money(new BigDecimal("10.00")).multiply(3);

    assertThat(result.getValue()).isEqualByComparingTo(new BigDecimal("30.00"));
  }

  @Test
  void multiplyByZeroReturnsZero() {
    var result = new Money(new BigDecimal("10.00")).multiply(0);

    assertThat(result.getValue()).isEqualByComparingTo(BigDecimal.ZERO);
  }

  @Test
  void allocateSplitsEvenlyWhenDivisible() {
    var parts = new Money(new BigDecimal("90.00")).allocate(3);

    assertThat(parts).hasSize(3);
    assertThat(parts.get(0).getValue()).isEqualByComparingTo(new BigDecimal("30.00"));
    assertThat(parts.get(1).getValue()).isEqualByComparingTo(new BigDecimal("30.00"));
    assertThat(parts.get(2).getValue()).isEqualByComparingTo(new BigDecimal("30.00"));
  }

  @Test
  void allocateDistributesRemainderHalfEven() {
    var parts = new Money(new BigDecimal("100.00")).allocate(3);

    var total = parts.stream().map(Money::getValue).reduce(BigDecimal.ZERO, BigDecimal::add);

    assertThat(parts).hasSize(3);
    assertThat(total).isEqualByComparingTo(new BigDecimal("100.00"));
    assertThat(parts.get(0).getValue()).isEqualByComparingTo(new BigDecimal("33.34"));
    assertThat(parts.get(1).getValue()).isEqualByComparingTo(new BigDecimal("33.33"));
    assertThat(parts.get(2).getValue()).isEqualByComparingTo(new BigDecimal("33.33"));
  }

  @Test
  void allocateSinglePartReturnsSameValue() {
    var parts = new Money(new BigDecimal("77.77")).allocate(1);

    assertThat(parts).hasSize(1);
    assertThat(parts.get(0).getValue()).isEqualByComparingTo(new BigDecimal("77.77"));
  }

  @Test
  void allocateRejectsNonPositiveParts() {
    assertThatThrownBy(() -> new Money(new BigDecimal("10.00")).allocate(0))
        .isInstanceOf(IllegalArgumentException.class);
  }

  @Test
  void isNegativeReturnsTrueForNegativeAmount() {
    assertThat(new Money(new BigDecimal("-1.00")).isNegative()).isTrue();
    assertThat(new Money(BigDecimal.ZERO).isNegative()).isFalse();
    assertThat(new Money(new BigDecimal("0.01")).isNegative()).isFalse();
  }

  @Test
  void sumOfEmptyListIsZero() {
    assertThat(Money.sum(java.util.List.of()).getValue()).isEqualByComparingTo(BigDecimal.ZERO);
  }

  @Test
  void sumAccumulatesValues() {
    var total =
        Money.sum(
            java.util.List.of(
                new Money(new BigDecimal("10.10")),
                new Money(new BigDecimal("20.20")),
                new Money(new BigDecimal("-5.00"))));

    assertThat(total.getValue()).isEqualByComparingTo(new BigDecimal("25.30"));
  }
}
