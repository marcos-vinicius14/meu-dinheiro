package com.marcos.meudinheiro.transaction.domain.valueobject;

import java.time.LocalDate;
import java.time.temporal.ChronoUnit;

public record DateInterval(LocalDate startDate, LocalDate endDate) {

    public DateInterval {
        if (startDate == null || endDate == null) {
            throw new IllegalArgumentException("Datas do intervalo são obrigatórias");
        }
        if (startDate.isAfter(endDate)) {
            throw new IllegalArgumentException("Data inicial não pode ser após a data final");
        }
    }

    public boolean contains(LocalDate date) {
        return !date.isBefore(startDate) && !date.isAfter(endDate);
    }

    public int daysRemaining(LocalDate from) {
        var days = (int) ChronoUnit.DAYS.between(from, endDate) + 1;
        return Math.max(1, days);
    }
}
