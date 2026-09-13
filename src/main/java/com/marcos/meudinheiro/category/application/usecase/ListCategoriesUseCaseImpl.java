package com.marcos.meudinheiro.category.application.usecase;

import com.marcos.meudinheiro.category.application.contract.ListCategoriesUseCase;
import com.marcos.meudinheiro.category.application.contract.dto.CategoryOutput;
import com.marcos.meudinheiro.category.application.mapper.CategoryMapper;
import com.marcos.meudinheiro.category.infraestructure.repository.CategoryRepository;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import org.springframework.stereotype.Service;

import java.util.List;
import java.util.UUID;

@Service
public class ListCategoriesUseCaseImpl implements ListCategoriesUseCase {

    private final CategoryRepository repository;

    public ListCategoriesUseCaseImpl(CategoryRepository repository) {
        this.repository = repository;
    }

    @Override
    public OperationResult<List<CategoryOutput>> execute(UUID userId) {
        var categories = repository.findAllByUserId(userId);

        var outputs = categories.stream()
                .map(CategoryMapper::toOutput)
                .toList();

        return OperationResult.success(outputs);
    }
}
