package com.marcos.meudinheiro.category.domain.valueobject;

import com.marcos.meudinheiro.shared.notification.ValidationResult;

public record CategoryDescription(String value) {

  public static final String DESCRIPTION_REQUIRED = "Descrição da categoria é obrigatória";
  public static final String DESCRIPTION_SIZE =
      "Descrição da categoria deve ter entre 3 e 100 caracteres";

  private static final int MIN_LENGTH = 3;
  private static final int MAX_LENGTH = 100;

  public static ValidationResult<CategoryDescription> create(String value) {
    if (value == null || value.isBlank()) {
      return ValidationResult.invalid(DESCRIPTION_REQUIRED);
    }

    var trimmed = value.trim();
    if (trimmed.length() < MIN_LENGTH || trimmed.length() > MAX_LENGTH) {
      return ValidationResult.invalid(DESCRIPTION_SIZE);
    }

    return ValidationResult.valid(new CategoryDescription(trimmed));
  }
}
