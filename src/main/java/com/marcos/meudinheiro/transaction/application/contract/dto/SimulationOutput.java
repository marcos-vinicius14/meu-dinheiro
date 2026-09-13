package com.marcos.meudinheiro.transaction.application.contract.dto;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.List;

public record SimulationOutput(
        List<CycleSimulationDto> cycles
) {

    public record CycleSimulationDto(
            LocalDate cycleStart,
            LocalDate cycleEnd,
            BigDecimal s2sToday,
            BigDecimal s2sReduction,
            BigDecimal s2sReductionPercent,
            BigDecimal projectedBalance,
            String healthStatus,
            boolean bottleneck
    ) {
    }
}
