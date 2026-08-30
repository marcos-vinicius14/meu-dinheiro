package com.marcos.meudinheiro.bankaccount.application.contract.constant;

public final class BankAccountMessages {

    private BankAccountMessages() {
    }

    public static final String ACCOUNT_NOT_FOUND = "Conta não encontrada";
    public static final String ACCOUNT_HAS_TRANSACTIONS = "Conta possui transações vinculadas";
    public static final String ACCOUNT_NAME_REQUIRED = "Nome da conta é obrigatório";
    public static final String ACCOUNT_NAME_SIZE = "Nome da conta deve ter entre 3 e 100 caracteres";
    public static final String ACCOUNT_TYPE_REQUIRED = "Tipo da conta é obrigatório";
    public static final String ACCOUNT_LIMIT_REACHED = "Limite de 3 contas atingido";
}
