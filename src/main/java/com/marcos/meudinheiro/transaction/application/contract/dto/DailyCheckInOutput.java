package com.marcos.meudinheiro.transaction.application.contract.dto;

import com.marcos.meudinheiro.transaction.domain.enums.HealthStatus;

import java.math.BigDecimal;

public record DailyCheckInOutput(
        BigDecimal s2sCalculated,
        BigDecimal spentToday,
        BigDecimal deltaFromSafeToSpend,
        HealthStatus healthStatus,
        BigDecimal nextDayS2S,
        BigDecimal projectedFreeBalance,
        int daysRemaining
) {
}
