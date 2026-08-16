package com.marcos.meudinheiro.shared.notification;

import java.util.List;

public record OperationResult(
        List<String> errors
) {

    public static OperationResult success() {
        return new OperationResult(List.of());
    }

    public static OperationResult failure(List<String> errors) {
        return new OperationResult(List.copyOf(errors));
    }

    public boolean isSuccess() {
        return errors.isEmpty();
    }

    public boolean isFailure() {
        return !isSuccess();
    }
}
