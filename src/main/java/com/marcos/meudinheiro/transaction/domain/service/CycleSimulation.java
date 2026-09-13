package com.marcos.meudinheiro.transaction.domain.service;

import com.marcos.meudinheiro.shared.valueobjects.Money;
import com.marcos.meudinheiro.transaction.domain.enums.HealthStatus;
import com.marcos.meudinheiro.transaction.domain.valueobject.DateInterval;

import java.math.BigDecimal;

public record CycleSimulation(
        DateInterval cycleInterval,
        EngineResult engineResult,
        Money s2sReduction,
        BigDecimal s2sReductionPercent,
        HealthStatus healthStatus,
        boolean bottleneck
) {
}
