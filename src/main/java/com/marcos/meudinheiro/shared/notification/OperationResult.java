package com.marcos.meudinheiro.shared.notification;

import java.util.List;

public record OperationResult<T>(
        T value,
        List<String> errors
) {

    public static OperationResult<Void> success() {
        return new OperationResult<>(null, List.of());
    }

    public static <T> OperationResult<T> success(T value) {
        return new OperationResult<>(value, List.of());
    }

    public static <T> OperationResult<T> failure(String error) {
        return failure(List.of(error));
    }

    public static <T> OperationResult<T> failure(List<String> errors) {
        return new OperationResult<>(null, List.copyOf(errors));
    }

    public boolean isSuccess() {
        return errors.isEmpty();
    }

    public boolean isFailure() {
        return !isSuccess();
    }
}
