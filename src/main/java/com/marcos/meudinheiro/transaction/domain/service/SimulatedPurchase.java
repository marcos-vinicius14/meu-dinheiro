package com.marcos.meudinheiro.transaction.domain.service;

import com.marcos.meudinheiro.shared.valueobjects.Money;
import java.time.LocalDate;

/** Compra simulada para o what-if: valor total, número de parcelas e data da primeira parcela. */
public record SimulatedPurchase(Money totalAmount, int installments, LocalDate firstDueDate) {

  public SimulatedPurchase {
    if (installments <= 0) {
      throw new IllegalArgumentException("Número de parcelas deve ser positivo");
    }
    totalAmount = totalAmount != null ? totalAmount : Money.zero();
  }
}
