package com.marcos.meudinheiro.category.application.usecase;

import com.marcos.meudinheiro.category.application.contract.UpdateCategoryUseCase;
import com.marcos.meudinheiro.category.application.contract.constant.CategoryMessages;
import com.marcos.meudinheiro.category.application.contract.dto.CategoryOutput;
import com.marcos.meudinheiro.category.application.contract.dto.UpdateCategoryInput;
import com.marcos.meudinheiro.category.application.mapper.CategoryMapper;
import com.marcos.meudinheiro.category.domain.model.CategoryModel;
import com.marcos.meudinheiro.category.domain.valueobject.CategoryDescription;
import com.marcos.meudinheiro.category.domain.valueobject.CategoryIcon;
import com.marcos.meudinheiro.category.infraestructure.repository.CategoryRepository;
import com.marcos.meudinheiro.shared.notification.Notification;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import java.util.UUID;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
public class UpdateCategoryUseCaseImpl implements UpdateCategoryUseCase {

  private final CategoryRepository repository;

  public UpdateCategoryUseCaseImpl(CategoryRepository repository) {
    this.repository = repository;
  }

  @Transactional
  @Override
  public OperationResult<CategoryOutput> execute(
      UUID userId, UUID categoryId, UpdateCategoryInput input) {
    repository.acquireCreationLock(creationLockId(userId));

    return repository
        .findById(categoryId)
        .filter(category -> category.belongsTo(userId))
        .map(category -> updateCategory(category, userId, input))
        .orElseGet(() -> OperationResult.failure(CategoryMessages.CATEGORY_NOT_FOUND));
  }

  private OperationResult<CategoryOutput> updateCategory(
      CategoryModel category, UUID userId, UpdateCategoryInput input) {
    var notification = new Notification();
    var description = notification.collect(CategoryDescription.create(input.description()));
    var icon = notification.collect(CategoryIcon.create(input.icon()));

    if (notification.hasErrors()) {
      return OperationResult.failure(notification.errors());
    }

    var newDescription = description.value();
    var changed = !newDescription.equalsIgnoreCase(category.getDescription());

    if (changed
        && repository.existsByUserIdAndDescriptionExcludingId(
            userId, newDescription, category.getId())) {
      return OperationResult.failure(CategoryMessages.CATEGORY_DUPLICATED);
    }

    category.updateDescription(description);
    category.updateIcon(icon);

    repository.save(category);

    return OperationResult.success(CategoryMapper.toOutput(category));
  }

  private static long creationLockId(UUID userId) {
    return userId.getMostSignificantBits() ^ userId.getLeastSignificantBits();
  }
}
