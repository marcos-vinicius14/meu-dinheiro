package com.marcos.meudinheiro.category.application.contract;

import com.marcos.meudinheiro.shared.notification.OperationResult;
import java.util.UUID;

public interface DeleteCategoryUseCase {
  OperationResult<Void> execute(UUID userId, UUID categoryId);
}
