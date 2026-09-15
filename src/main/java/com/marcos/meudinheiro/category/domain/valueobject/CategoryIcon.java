package com.marcos.meudinheiro.category.domain.valueobject;

import com.marcos.meudinheiro.shared.notification.ValidationResult;

public record CategoryIcon(String value) {

  public static final String ICON_SIZE = "Ícone da categoria deve ter no máximo 255 caracteres";

  private static final int MAX_LENGTH = 255;

  @SuppressWarnings("NullAway")
  public static ValidationResult<CategoryIcon> create(String value) {
    if (value == null || value.isBlank()) {
      return ValidationResult.valid(null);
    }

    var trimmed = value.trim();
    if (trimmed.length() > MAX_LENGTH) {
      return ValidationResult.invalid(ICON_SIZE);
    }

    return ValidationResult.valid(new CategoryIcon(trimmed));
  }
}
