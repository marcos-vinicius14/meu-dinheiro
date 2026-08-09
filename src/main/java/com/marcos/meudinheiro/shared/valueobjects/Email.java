package com.marcos.meudinheiro.shared.valueobjects;

import jakarta.persistence.Column;
import jakarta.persistence.Embeddable;

import java.util.Locale;
import java.util.Optional;
import java.util.regex.Pattern;

@Embeddable
public class Email {
    private static final int MAX_LENGTH = 255;

    private static final Pattern EMAIL_PATTERN =
            Pattern.compile("^[A-Z0-9._%+-]+@[A-Z0-9.-]+\\.[A-Z]{2,}$",
                    Pattern.CASE_INSENSITIVE);

    @Column(name = "email", unique = true)
    private String value;

    protected Email() {}

    public Email(String email) {
        this.value = validate(email);
    }

    private static String validate(String value) {
        return Optional.ofNullable(value)
                .map(String::trim)
                .map(email -> email.toLowerCase(Locale.ROOT))
                .filter(email -> !email.isBlank())
                .filter(email -> email.length() <= MAX_LENGTH)
                .filter(email -> EMAIL_PATTERN.matcher(email).matches())
                .orElseThrow(() -> new IllegalArgumentException("Email invalido"));
    }

    public String value() {
        return value;
    }
}
