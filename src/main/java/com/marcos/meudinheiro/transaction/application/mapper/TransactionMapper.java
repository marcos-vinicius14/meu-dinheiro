package com.marcos.meudinheiro.transaction.application.mapper;

import com.marcos.meudinheiro.transaction.application.contract.dto.TransactionOutput;
import com.marcos.meudinheiro.transaction.domain.model.TransactionModel;

public final class TransactionMapper {

  private TransactionMapper() {}

  public static TransactionOutput toOutput(TransactionModel transaction) {
    return new TransactionOutput(
        transaction.getId(),
        transaction.getDescription(),
        transaction.getAmount().getValue(),
        transaction.getType(),
        transaction.getStatus(),
        transaction.getCategory() != null ? transaction.getCategory().getId() : null,
        transaction.getBankAccount() != null ? transaction.getBankAccount().getId() : null,
        transaction.getDueDate(),
        transaction.getPaymentDate(),
        transaction.getBundleId(),
        transaction.getInstallmentNumber(),
        transaction.getTotalInstallments());
  }
}
