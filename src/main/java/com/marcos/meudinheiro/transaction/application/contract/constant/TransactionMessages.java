package com.marcos.meudinheiro.transaction.application.contract.constant;

public final class TransactionMessages {

  private TransactionMessages() {}

  public static final String TRANSACTION_NOT_FOUND = "Transação não encontrada";
  public static final String TRANSACTION_NOT_FOUND_OR_NOT_OWNED = "Transação não encontrada";
  public static final String TRANSACTION_AMOUNT_REQUIRED = "Valor da transação deve ser positivo";
  public static final String TRANSACTION_DUE_DATE_REQUIRED = "Data de vencimento é obrigatória";
  public static final String TRANSACTION_TYPE_REQUIRED = "Tipo da transação é obrigatório";
  public static final String TRANSACTION_CATEGORY_REQUIRED = "Categoria é obrigatória";
  public static final String TRANSACTION_CATEGORY_NOT_FOUND = "Categoria não encontrada";
  public static final String TRANSACTION_CATEGORY_INVALID_TYPE =
      "Categoria incompatível com o tipo da transação";
  public static final String TRANSACTION_BANK_ACCOUNT_NOT_FOUND = "Conta bancária não encontrada";
  public static final String TRANSACTION_TYPE_NOT_INSTALLMENT =
      "Transações parceladas devem ser criadas via bundle";
  public static final String BUNDLE_INSTALLMENTS_REQUIRED = "Número de parcelas deve ser positivo";
  public static final String BUNDLE_TOTAL_AMOUNT_REQUIRED =
      "Valor total do parcelamento deve ser positivo";
  public static final String BUNDLE_FIRST_DUE_DATE_REQUIRED =
      "Data do primeiro vencimento é obrigatória";
  public static final String BUNDLE_NOT_FOUND = "Parcelamento não encontrado";
}
