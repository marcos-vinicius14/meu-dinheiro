package com.marcos.meudinheiro.category.application.usecase;

import com.marcos.meudinheiro.category.application.contract.CategoryUserResolver;
import com.marcos.meudinheiro.category.application.contract.CreateCategoryUseCase;
import com.marcos.meudinheiro.category.application.contract.constant.CategoryMessages;
import com.marcos.meudinheiro.category.application.contract.dto.CategoryOutput;
import com.marcos.meudinheiro.category.application.contract.dto.CreateCategoryInput;
import com.marcos.meudinheiro.category.application.mapper.CategoryMapper;
import com.marcos.meudinheiro.category.domain.model.CategoryModel;
import com.marcos.meudinheiro.category.domain.valueobject.CategoryDescription;
import com.marcos.meudinheiro.category.domain.valueobject.CategoryIcon;
import com.marcos.meudinheiro.category.infraestructure.repository.CategoryRepository;
import com.marcos.meudinheiro.shared.notification.Notification;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.shared.notification.ValidationResult;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.UUID;

@Service
public class CreateCategoryUseCaseImpl implements CreateCategoryUseCase {

    private final CategoryRepository repository;
    private final CategoryUserResolver userResolver;

    public CreateCategoryUseCaseImpl(
            CategoryRepository repository,
            CategoryUserResolver userResolver
    ) {
        this.repository = repository;
        this.userResolver = userResolver;
    }

    @Transactional
    @Override
    public OperationResult<CategoryOutput> execute(UUID userId, CreateCategoryInput input) {
        repository.acquireCreationLock(creationLockId(userId));

        var validation = validateCategory(userId, input);

        if (validation.isInvalid()) {
            return OperationResult.failure(validation.errors());
        }

        var category = CategoryModel.create(
                validation.value().description(),
                validation.value().icon(),
                userResolver.resolve(userId)
        );

        repository.save(category);

        return OperationResult.success(CategoryMapper.toOutput(category));
    }

    private ValidationResult<CategoryValidation> validateCategory(UUID userId, CreateCategoryInput input) {
        var notification = new Notification();

        var description = notification.collect(
                CategoryDescription.create(input.description())
        );

        var icon = notification.collect(
                CategoryIcon.create(input.icon())
        );

        if (description != null && repository.existsByUserIdAndDescriptionIgnoreCase(userId, description.value())) {
            notification.collect(ValidationResult.invalid(CategoryMessages.CATEGORY_DUPLICATED));
        }

        if (notification.hasErrors()) {
            return ValidationResult.invalid(notification.errors());
        }

        return ValidationResult.valid(new CategoryValidation(description, icon));
    }

    private static long creationLockId(UUID userId) {
        return userId.getMostSignificantBits() ^ userId.getLeastSignificantBits();
    }

    private record CategoryValidation(CategoryDescription description, CategoryIcon icon) {
    }
}
