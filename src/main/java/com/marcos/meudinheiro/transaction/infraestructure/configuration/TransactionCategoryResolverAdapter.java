package com.marcos.meudinheiro.transaction.infraestructure.configuration;

import com.marcos.meudinheiro.category.domain.model.CategoryModel;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.transaction.application.contract.TransactionCategoryResolver;
import com.marcos.meudinheiro.transaction.application.contract.constant.TransactionMessages;
import com.marcos.meudinheiro.category.infraestructure.repository.CategoryRepository;
import org.springframework.stereotype.Component;

import java.util.UUID;

@Component
public class TransactionCategoryResolverAdapter implements TransactionCategoryResolver {

    private final CategoryRepository categoryRepository;

    public TransactionCategoryResolverAdapter(CategoryRepository categoryRepository) {
        this.categoryRepository = categoryRepository;
    }

    @Override
    public OperationResult<CategoryModel> resolve(UUID userId, UUID categoryId) {
        if (categoryId == null) {
            return OperationResult.failure(TransactionMessages.TRANSACTION_CATEGORY_REQUIRED);
        }

        return categoryRepository.findById(categoryId)
                .filter(category -> category.belongsTo(userId))
                .map(OperationResult::success)
                .orElseGet(() -> OperationResult.failure(TransactionMessages.TRANSACTION_CATEGORY_NOT_FOUND));
    }
}
