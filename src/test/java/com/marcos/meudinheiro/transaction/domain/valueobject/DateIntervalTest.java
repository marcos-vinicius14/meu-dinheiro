package com.marcos.meudinheiro.transaction.domain.valueobject;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import java.time.LocalDate;
import org.junit.jupiter.api.Test;

class DateIntervalTest {

  @Test
  void createsValidInterval() {
    var interval = new DateInterval(LocalDate.of(2026, 9, 1), LocalDate.of(2026, 9, 30));

    assertThat(interval.startDate()).isEqualTo(LocalDate.of(2026, 9, 1));
    assertThat(interval.endDate()).isEqualTo(LocalDate.of(2026, 9, 30));
  }

  @Test
  void rejectsStartAfterEnd() {
    assertThatThrownBy(() -> new DateInterval(LocalDate.of(2026, 9, 30), LocalDate.of(2026, 9, 1)))
        .isInstanceOf(IllegalArgumentException.class);
  }

  @Test
  void acceptsSingleDayInterval() {
    var date = LocalDate.of(2026, 9, 15);
    var interval = new DateInterval(date, date);

    assertThat(interval.daysRemaining(date)).isEqualTo(1);
  }

  @Test
  void daysRemainingIsInclusiveOfCurrentDay() {
    var interval = new DateInterval(LocalDate.of(2026, 9, 1), LocalDate.of(2026, 9, 30));

    assertThat(interval.daysRemaining(LocalDate.of(2026, 9, 13))).isEqualTo(18);
  }

  @Test
  void daysRemainingNeverBelowOne() {
    var interval = new DateInterval(LocalDate.of(2026, 9, 1), LocalDate.of(2026, 9, 30));

    assertThat(interval.daysRemaining(LocalDate.of(2026, 10, 5))).isEqualTo(1);
  }

  @Test
  void daysRemainingOnLastDayIsOne() {
    var interval = new DateInterval(LocalDate.of(2026, 9, 1), LocalDate.of(2026, 9, 30));

    assertThat(interval.daysRemaining(LocalDate.of(2026, 9, 30))).isEqualTo(1);
  }
}
