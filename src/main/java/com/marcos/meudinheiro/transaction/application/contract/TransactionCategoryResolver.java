package com.marcos.meudinheiro.transaction.application.contract;

import com.marcos.meudinheiro.category.domain.model.CategoryModel;
import com.marcos.meudinheiro.shared.notification.OperationResult;

import java.util.UUID;

/**
 * Porta de acesso à categoria: valida ownership sem acessar repository alheio.
 */
public interface TransactionCategoryResolver {
    OperationResult<CategoryModel> resolve(UUID userId, UUID categoryId);
}
