package com.marcos.meudinheiro.category.application.usecase;

import com.marcos.meudinheiro.category.application.contract.FindCategoryUseCase;
import com.marcos.meudinheiro.category.application.contract.constant.CategoryMessages;
import com.marcos.meudinheiro.category.application.contract.dto.CategoryOutput;
import com.marcos.meudinheiro.category.application.mapper.CategoryMapper;
import com.marcos.meudinheiro.category.infraestructure.repository.CategoryRepository;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import java.util.UUID;
import org.springframework.stereotype.Service;

@Service
public class FindCategoryUseCaseImpl implements FindCategoryUseCase {

  private final CategoryRepository repository;

  public FindCategoryUseCaseImpl(CategoryRepository repository) {
    this.repository = repository;
  }

  @Override
  public OperationResult<CategoryOutput> execute(UUID userId, UUID categoryId) {
    return repository
        .findById(categoryId)
        .filter(category -> category.belongsTo(userId))
        .map(CategoryMapper::toOutput)
        .map(OperationResult::success)
        .orElseGet(() -> OperationResult.failure(CategoryMessages.CATEGORY_NOT_FOUND));
  }
}
