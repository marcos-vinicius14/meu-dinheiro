package com.marcos.meudinheiro.user.domain.valueobject;

import com.marcos.meudinheiro.shared.notification.ValidationResult;
import jakarta.persistence.Column;
import jakarta.persistence.Embeddable;
import java.util.Locale;
import java.util.Optional;
import java.util.regex.Pattern;

@Embeddable
public class Email {
  private static final int MAX_LENGTH = 255;

  private static final Pattern EMAIL_PATTERN =
      Pattern.compile("^[A-Z0-9._%+-]+@[A-Z0-9.-]+\\.[A-Z]{2,}$", Pattern.CASE_INSENSITIVE);

  @Column(name = "email", unique = true)
  private String value;

  protected Email() {}

  private Email(String email) {
    this.value = email;
  }

  public static ValidationResult<Email> create(String value) {
    return Optional.ofNullable(value)
        .map(String::trim)
        .map(email -> email.toLowerCase(Locale.ROOT))
        .filter(email -> !email.isBlank())
        .filter(email -> email.length() <= MAX_LENGTH)
        .filter(email -> EMAIL_PATTERN.matcher(email).matches())
        .<ValidationResult<Email>>map(email -> ValidationResult.valid(new Email(email)))
        .orElseGet(() -> ValidationResult.invalid("Email inválido"));
  }

  public String value() {
    return value;
  }
}
