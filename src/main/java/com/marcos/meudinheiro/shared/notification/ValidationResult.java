package com.marcos.meudinheiro.shared.notification;

import java.util.List;

public record ValidationResult<T>(T value, List<String> errors) {
  public static <T> ValidationResult<T> valid(T value) {
    return new ValidationResult<>(value, List.of());
  }

  @SuppressWarnings("NullAway")
  public static <T> ValidationResult<T> invalid(String error) {
    return new ValidationResult<>(null, List.of(error));
  }

  @SuppressWarnings("NullAway")
  public static <T> ValidationResult<T> invalid(List<String> errors) {
    return new ValidationResult<>(null, List.copyOf(errors));
  }

  public boolean isValid() {
    return errors.isEmpty();
  }

  public boolean isInvalid() {
    return !isValid();
  }
}
