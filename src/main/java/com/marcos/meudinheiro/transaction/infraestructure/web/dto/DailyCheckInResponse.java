package com.marcos.meudinheiro.transaction.infraestructure.web.dto;

import com.marcos.meudinheiro.transaction.application.contract.dto.DailyCheckInOutput;

import java.math.BigDecimal;

public record DailyCheckInResponse(
        BigDecimal s2sCalculated,
        BigDecimal spentToday,
        BigDecimal deltaFromSafeToSpend,
        String healthStatus,
        BigDecimal nextDayS2S,
        BigDecimal projectedFreeBalance,
        int daysRemaining
) {

    public static DailyCheckInResponse from(DailyCheckInOutput output) {
        return new DailyCheckInResponse(
                output.s2sCalculated(),
                output.spentToday(),
                output.deltaFromSafeToSpend(),
                output.healthStatus().name(),
                output.nextDayS2S(),
                output.projectedFreeBalance(),
                output.daysRemaining()
        );
    }
}
