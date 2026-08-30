package com.marcos.meudinheiro.bankaccount.infraestructure.web.controller;

import com.marcos.meudinheiro.bankaccount.application.contract.CreateBankAccountUseCase;
import com.marcos.meudinheiro.bankaccount.application.contract.DeleteBankAccountUseCase;
import com.marcos.meudinheiro.bankaccount.application.contract.FindBankAccountUseCase;
import com.marcos.meudinheiro.bankaccount.application.contract.ListBankAccountsUseCase;
import com.marcos.meudinheiro.bankaccount.application.contract.UpdateBankAccountUseCase;
import com.marcos.meudinheiro.bankaccount.application.contract.constant.BankAccountMessages;
import com.marcos.meudinheiro.bankaccount.application.contract.dto.CreateBankAccountInput;
import com.marcos.meudinheiro.bankaccount.application.contract.dto.UpdateBankAccountInput;
import com.marcos.meudinheiro.bankaccount.application.mapper.BankAccountMapper;
import com.marcos.meudinheiro.bankaccount.infraestructure.web.dto.BankAccountResponse;
import com.marcos.meudinheiro.bankaccount.infraestructure.web.dto.CreateBankAccountRequest;
import com.marcos.meudinheiro.bankaccount.infraestructure.web.dto.UpdateBankAccountRequest;
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
@RequestMapping("/bankaccounts")
public class BankAccountController {

    private final CreateBankAccountUseCase createUseCase;
    private final ListBankAccountsUseCase listUseCase;
    private final FindBankAccountUseCase findUseCase;
    private final UpdateBankAccountUseCase updateUseCase;
    private final DeleteBankAccountUseCase deleteUseCase;
    private final CurrentIdentity currentIdentity;

    public BankAccountController(
            CreateBankAccountUseCase createUseCase,
            ListBankAccountsUseCase listUseCase,
            FindBankAccountUseCase findUseCase,
            UpdateBankAccountUseCase updateUseCase,
            DeleteBankAccountUseCase deleteUseCase,
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
            @Valid @RequestBody CreateBankAccountRequest request
    ) {
        var userId = currentIdentity.findCurrentAuthenticadedUser();
        var input = new CreateBankAccountInput(
                request.name(),
                request.type(),
                request.initialBalance()
        );

        var result = createUseCase.execute(userId, input);

        if (result.isFailure()) {
            return ResponseEntity
                    .badRequest()
                    .body(new ErrorResponse(result.errors()));
        }

        return ResponseEntity
                .status(HttpStatus.CREATED)
                .body(BankAccountMapper.toResponse(result.value()));
    }

    @GetMapping
    ResponseEntity<List<BankAccountResponse>> list() {
        var userId = currentIdentity.findCurrentAuthenticadedUser();
        var result = listUseCase.execute(userId);

        var responses = result.value().stream()
                .map(BankAccountMapper::toResponse)
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

        return ResponseEntity.ok(BankAccountMapper.toResponse(result.value()));
    }

    @PutMapping("/{id}")
    ResponseEntity<Object> update(
            @PathVariable UUID id,
            @Valid @RequestBody UpdateBankAccountRequest request
    ) {
        var userId = currentIdentity.findCurrentAuthenticadedUser();
        var input = new UpdateBankAccountInput(
                request.name(),
                request.initialBalance()
        );

        var result = updateUseCase.execute(userId, id, input);

        if (result.isFailure()) {
            var error = result.errors().getFirst();
            if (BankAccountMessages.ACCOUNT_NOT_FOUND.equals(error)) {
                return ResponseEntity
                        .status(HttpStatus.NOT_FOUND)
                        .body(new ErrorResponse(result.errors()));
            }

            return ResponseEntity
                    .badRequest()
                    .body(new ErrorResponse(result.errors()));
        }

        return ResponseEntity.ok(BankAccountMapper.toResponse(result.value()));
    }

    @DeleteMapping("/{id}")
    ResponseEntity<Object> delete(@PathVariable UUID id) {
        var userId = currentIdentity.findCurrentAuthenticadedUser();
        var result = deleteUseCase.execute(userId, id);

        if (result.isFailure()) {
            var error = result.errors().getFirst();
            if (BankAccountMessages.ACCOUNT_HAS_TRANSACTIONS.equals(error)) {
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
