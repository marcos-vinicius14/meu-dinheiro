package com.marcos.meudinheiro.category.application.contract;

import com.marcos.meudinheiro.category.application.contract.dto.CategoryOutput;
import com.marcos.meudinheiro.category.application.contract.dto.UpdateCategoryInput;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import java.util.UUID;

public interface UpdateCategoryUseCase {
  OperationResult<CategoryOutput> execute(UUID userId, UUID categoryId, UpdateCategoryInput input);
}
