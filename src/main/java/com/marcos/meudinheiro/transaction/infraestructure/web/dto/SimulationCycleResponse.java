package com.marcos.meudinheiro.transaction.infraestructure.web.dto;

import java.math.BigDecimal;
import java.time.LocalDate;

public record SimulationCycleResponse(
    LocalDate cycleStart,
    LocalDate cycleEnd,
    BigDecimal s2sToday,
    BigDecimal s2sReduction,
    BigDecimal s2sReductionPercent,
    BigDecimal projectedBalance,
    String healthStatus,
    boolean bottleneck) {}
