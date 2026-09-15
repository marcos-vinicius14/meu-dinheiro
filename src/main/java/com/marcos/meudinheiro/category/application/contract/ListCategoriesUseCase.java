package com.marcos.meudinheiro.category.application.contract;

import com.marcos.meudinheiro.category.application.contract.dto.CategoryOutput;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import java.util.List;
import java.util.UUID;

public interface ListCategoriesUseCase {
  OperationResult<List<CategoryOutput>> execute(UUID userId);
}
