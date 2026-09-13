package com.marcos.meudinheiro.shared.valueobjects;

import jakarta.persistence.Column;
import jakarta.persistence.Embeddable;

import java.math.BigDecimal;
import java.math.RoundingMode;
import java.util.List;

@Embeddable
public class Money {

    private static final int SCALE = 2;
    private static final BigDecimal HUNDRED = BigDecimal.valueOf(100);

    @Column(precision = 19, scale = 2)
    private BigDecimal value;

    protected Money() {}

    public Money(BigDecimal value) {
        this.value = value.setScale(SCALE, RoundingMode.HALF_EVEN);
    }

    public BigDecimal getValue() {
        return value;
    }

    public Money add(Money other) {
        return new Money(value.add(other.value));
    }

    public Money subtract(Money other) {
        return new Money(value.subtract(other.value));
    }

    public Money multiply(int factor) {
        return new Money(value.multiply(BigDecimal.valueOf(factor)));
    }

    public List<Money> allocate(int parts) {
        if (parts <= 0) {
            throw new IllegalArgumentException("Número de partes deve ser positivo");
        }

        var part = new Money(value.divide(BigDecimal.valueOf(parts), SCALE, RoundingMode.HALF_EVEN));
        var remainder = new Money(value.subtract(part.getValue().multiply(BigDecimal.valueOf(parts - 1))));

        var allocations = new java.util.ArrayList<Money>(parts);
        allocations.add(remainder);
        for (int i = 1; i < parts; i++) {
            allocations.add(part);
        }
        return allocations;
    }

    public boolean isNegative() {
        return value.signum() < 0;
    }

    public static Money sum(List<Money> values) {
        return values.stream()
                .reduce(new Money(BigDecimal.ZERO), Money::add);
    }

    public static Money fromCents(long cents) {
        return new Money(BigDecimal.valueOf(cents, SCALE));
    }

    public long toCents() {
        return value.movePointRight(SCALE).longValueExact();
    }

    public static Money ofRatio(Money base, BigDecimal ratio) {
        return new Money(base.value.multiply(ratio).setScale(SCALE, RoundingMode.HALF_EVEN));
    }

    public static Money min(Money a, Money b) {
        return a.value.compareTo(b.value) <= 0 ? a : b;
    }

    public static Money max(Money a, Money b) {
        return a.value.compareTo(b.value) >= 0 ? a : b;
    }

    public static Money zero() {
        return new Money(BigDecimal.ZERO);
    }

    public boolean isLessThan(Money other) {
        return value.compareTo(other.value) < 0;
    }

    public boolean isGreaterThan(Money other) {
        return value.compareTo(other.value) > 0;
    }

    public boolean isZero() {
        return value.signum() == 0;
    }

    public BigDecimal ratioOver(Money base) {
        return value.divide(base.value, 6, RoundingMode.HALF_EVEN);
    }

    public BigDecimal percentOver(Money base) {
        return ratioOver(base).multiply(HUNDRED).setScale(2, RoundingMode.HALF_EVEN);
    }
}
