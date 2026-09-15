package com.marcos.meudinheiro.category.application.contract;

import com.marcos.meudinheiro.category.application.contract.dto.CategoryOutput;
import com.marcos.meudinheiro.category.application.contract.dto.CreateCategoryInput;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import java.util.UUID;

public interface CreateCategoryUseCase {
  OperationResult<CategoryOutput> execute(UUID userId, CreateCategoryInput input);
}
