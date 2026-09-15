package com.marcos.meudinheiro.category.application.usecase;

import com.marcos.meudinheiro.category.application.contract.DeleteCategoryUseCase;
import com.marcos.meudinheiro.category.application.contract.constant.CategoryMessages;
import com.marcos.meudinheiro.category.infraestructure.repository.CategoryRepository;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import java.util.UUID;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
public class DeleteCategoryUseCaseImpl implements DeleteCategoryUseCase {

  private final CategoryRepository repository;

  public DeleteCategoryUseCaseImpl(CategoryRepository repository) {
    this.repository = repository;
  }

  @Transactional
  @Override
  public OperationResult<Void> execute(UUID userId, UUID categoryId) {
    return repository
        .findById(categoryId)
        .filter(category -> category.belongsTo(userId))
        .map(category -> deleteCategory(categoryId))
        .orElseGet(() -> OperationResult.failure(CategoryMessages.CATEGORY_NOT_FOUND));
  }

  private OperationResult<Void> deleteCategory(UUID categoryId) {
    var deletedRows = repository.deleteIfNoTransactions(categoryId);

    if (deletedRows == 0) {
      return OperationResult.failure(CategoryMessages.CATEGORY_HAS_TRANSACTIONS);
    }

    return OperationResult.success();
  }
}
