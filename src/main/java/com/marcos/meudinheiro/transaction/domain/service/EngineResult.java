package com.marcos.meudinheiro.transaction.domain.service;

import com.marcos.meudinheiro.shared.valueobjects.Money;
import com.marcos.meudinheiro.transaction.domain.enums.HealthStatus;

public record EngineResult(
    Money s2sToday,
    Money remainingFlexible,
    Money projectedBalance,
    Money netLiquidityBeforeFlex,
    Money flexibleSpent,
    Money projectedIncome,
    Money committedExpenses,
    HealthStatus healthStatus,
    int daysRemaining) {}
