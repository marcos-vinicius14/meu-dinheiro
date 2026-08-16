package com.marcos.meudinheiro.shared.notification;

import java.util.ArrayList;
import java.util.List;

public final class Notification {

    private final List<String> errors = new ArrayList<>();

    public <T> T collect(ValidationResult<T> result) {
        errors.addAll(result.errors());
        return result.value();
    }

    public boolean hasErrors() {
        return !errors.isEmpty();
    }

    public List<String> errors() {
        return List.copyOf(errors);
    }
}
