package com.marcos.meudinheiro.category.infraestructure.web.controller;

import com.marcos.meudinheiro.category.application.contract.CreateCategoryUseCase;
import com.marcos.meudinheiro.category.application.contract.DeleteCategoryUseCase;
import com.marcos.meudinheiro.category.application.contract.FindCategoryUseCase;
import com.marcos.meudinheiro.category.application.contract.ListCategoriesUseCase;
import com.marcos.meudinheiro.category.application.contract.UpdateCategoryUseCase;
import com.marcos.meudinheiro.category.application.contract.constant.CategoryMessages;
import com.marcos.meudinheiro.category.application.contract.dto.CreateCategoryInput;
import com.marcos.meudinheiro.category.application.contract.dto.UpdateCategoryInput;
import com.marcos.meudinheiro.category.infraestructure.web.dto.CategoryResponse;
import com.marcos.meudinheiro.category.infraestructure.web.dto.CreateCategoryRequest;
import com.marcos.meudinheiro.category.infraestructure.web.dto.UpdateCategoryRequest;
import com.marcos.meudinheiro.identity.application.contract.CurrentIdentity;
import com.marcos.meudinheiro.shared.infraestructure.web.response.ErrorResponse;
import jakarta.validation.Valid;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;
import java.util.UUID;

@RestController
@RequestMapping("/categories")
public class CategoryController {

    private final CreateCategoryUseCase createUseCase;
    private final ListCategoriesUseCase listUseCase;
    private final FindCategoryUseCase findUseCase;
    private final UpdateCategoryUseCase updateUseCase;
    private final DeleteCategoryUseCase deleteUseCase;
    private final CurrentIdentity currentIdentity;

    public CategoryController(
            CreateCategoryUseCase createUseCase,
            ListCategoriesUseCase listUseCase,
            FindCategoryUseCase findUseCase,
            UpdateCategoryUseCase updateUseCase,
            DeleteCategoryUseCase deleteUseCase,
            CurrentIdentity currentIdentity
    ) {
        this.createUseCase = createUseCase;
        this.listUseCase = listUseCase;
        this.findUseCase = findUseCase;
        this.updateUseCase = updateUseCase;
        this.deleteUseCase = deleteUseCase;
        this.currentIdentity = currentIdentity;
    }

    @PostMapping
    ResponseEntity<Object> create(
            @Valid @RequestBody CreateCategoryRequest request
    ) {
        var userId = currentIdentity.findCurrentAuthenticadedUser();
        var input = new CreateCategoryInput(
                request.description(),
                request.icon(),
                request.isFlexibleOrDefault()
        );

        var result = createUseCase.execute(userId, input);

        if (result.isFailure()) {
            return ResponseEntity
                    .badRequest()
                    .body(new ErrorResponse(result.errors()));
        }

        return ResponseEntity
                .status(HttpStatus.CREATED)
                .body(CategoryResponse.from(result.value()));
    }

    @GetMapping
    ResponseEntity<List<CategoryResponse>> list() {
        var userId = currentIdentity.findCurrentAuthenticadedUser();
        var result = listUseCase.execute(userId);

        var responses = result.value().stream()
                .map(CategoryResponse::from)
                .toList();

        return ResponseEntity.ok(responses);
    }

    @GetMapping("/{id}")
    ResponseEntity<Object> find(@PathVariable UUID id) {
        var userId = currentIdentity.findCurrentAuthenticadedUser();
        var result = findUseCase.execute(userId, id);

        if (result.isFailure()) {
            return ResponseEntity
                    .status(HttpStatus.NOT_FOUND)
                    .body(new ErrorResponse(result.errors()));
        }

        return ResponseEntity.ok(CategoryResponse.from(result.value()));
    }

    @PutMapping("/{id}")
    ResponseEntity<Object> update(
            @PathVariable UUID id,
            @Valid @RequestBody UpdateCategoryRequest request
    ) {
        var userId = currentIdentity.findCurrentAuthenticadedUser();
        var input = new UpdateCategoryInput(
                request.description(),
                request.icon()
        );

        var result = updateUseCase.execute(userId, id, input);

        if (result.isFailure()) {
            var error = result.errors().getFirst();
            if (CategoryMessages.CATEGORY_NOT_FOUND.equals(error)) {
                return ResponseEntity
                        .status(HttpStatus.NOT_FOUND)
                        .body(new ErrorResponse(result.errors()));
            }

            return ResponseEntity
                    .badRequest()
                    .body(new ErrorResponse(result.errors()));
        }

        return ResponseEntity.ok(CategoryResponse.from(result.value()));
    }

    @DeleteMapping("/{id}")
    ResponseEntity<Object> delete(@PathVariable UUID id) {
        var userId = currentIdentity.findCurrentAuthenticadedUser();
        var result = deleteUseCase.execute(userId, id);

        if (result.isFailure()) {
            var error = result.errors().getFirst();
            if (CategoryMessages.CATEGORY_HAS_TRANSACTIONS.equals(error)) {
                return ResponseEntity
                        .status(HttpStatus.CONFLICT)
                        .body(new ErrorResponse(result.errors()));
            }

            return ResponseEntity
                    .status(HttpStatus.NOT_FOUND)
                    .body(new ErrorResponse(result.errors()));
        }

        return ResponseEntity.noContent().build();
    }
}
