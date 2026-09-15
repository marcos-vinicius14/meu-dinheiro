package com.marcos.meudinheiro.transaction.infraestructure.web.controller;

import com.marcos.meudinheiro.identity.application.contract.CurrentIdentity;
import com.marcos.meudinheiro.shared.infraestructure.web.response.ErrorResponse;
import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.transaction.application.contract.CreateTransactionBundleUseCase;
import com.marcos.meudinheiro.transaction.application.contract.CreateTransactionUseCase;
import com.marcos.meudinheiro.transaction.application.contract.DailyCheckInUseCase;
import com.marcos.meudinheiro.transaction.application.contract.DeleteTransactionUseCase;
import com.marcos.meudinheiro.transaction.application.contract.FindTransactionUseCase;
import com.marcos.meudinheiro.transaction.application.contract.ListTransactionsUseCase;
import com.marcos.meudinheiro.transaction.application.contract.SimulatePurchaseUseCase;
import com.marcos.meudinheiro.transaction.application.contract.UpdateTransactionUseCase;
import com.marcos.meudinheiro.transaction.application.contract.dto.CreateTransactionBundleInput;
import com.marcos.meudinheiro.transaction.application.contract.dto.CreateTransactionInput;
import com.marcos.meudinheiro.transaction.application.contract.dto.DailyCheckInInput;
import com.marcos.meudinheiro.transaction.application.contract.dto.SimulationInput;
import com.marcos.meudinheiro.transaction.application.contract.dto.UpdateTransactionInput;
import com.marcos.meudinheiro.transaction.domain.enums.TransactionType;
import com.marcos.meudinheiro.transaction.infraestructure.web.dto.CreateBundleRequest;
import com.marcos.meudinheiro.transaction.infraestructure.web.dto.CreateTransactionRequest;
import com.marcos.meudinheiro.transaction.infraestructure.web.dto.DailyCheckInRequest;
import com.marcos.meudinheiro.transaction.infraestructure.web.dto.DailyCheckInResponse;
import com.marcos.meudinheiro.transaction.infraestructure.web.dto.SimulationCycleResponse;
import com.marcos.meudinheiro.transaction.infraestructure.web.dto.SimulationRequest;
import com.marcos.meudinheiro.transaction.infraestructure.web.dto.TransactionBundleResponse;
import com.marcos.meudinheiro.transaction.infraestructure.web.dto.TransactionResponse;
import com.marcos.meudinheiro.transaction.infraestructure.web.dto.UpdateTransactionRequest;
import jakarta.validation.Valid;
import java.util.List;
import java.util.UUID;
import org.jspecify.annotations.Nullable;
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

@RestController
@RequestMapping("/transactions")
public class TransactionController {

  private final CreateTransactionUseCase createUseCase;
  private final ListTransactionsUseCase listUseCase;
  private final FindTransactionUseCase findUseCase;
  private final UpdateTransactionUseCase updateUseCase;
  private final DeleteTransactionUseCase deleteUseCase;
  private final CreateTransactionBundleUseCase bundleUseCase;
  private final DailyCheckInUseCase checkInUseCase;
  private final SimulatePurchaseUseCase simulationUseCase;
  private final CurrentIdentity currentIdentity;

  public TransactionController(
      CreateTransactionUseCase createUseCase,
      ListTransactionsUseCase listUseCase,
      FindTransactionUseCase findUseCase,
      UpdateTransactionUseCase updateUseCase,
      DeleteTransactionUseCase deleteUseCase,
      CreateTransactionBundleUseCase bundleUseCase,
      DailyCheckInUseCase checkInUseCase,
      SimulatePurchaseUseCase simulationUseCase,
      CurrentIdentity currentIdentity) {
    this.createUseCase = createUseCase;
    this.listUseCase = listUseCase;
    this.findUseCase = findUseCase;
    this.updateUseCase = updateUseCase;
    this.deleteUseCase = deleteUseCase;
    this.bundleUseCase = bundleUseCase;
    this.checkInUseCase = checkInUseCase;
    this.simulationUseCase = simulationUseCase;
    this.currentIdentity = currentIdentity;
  }

  @PostMapping
  ResponseEntity<Object> create(@Valid @RequestBody CreateTransactionRequest request) {
    var userId = currentIdentity.findCurrentAuthenticadedUser();
    var input =
        new CreateTransactionInput(
            request.description(),
            request.amount(),
            parseType(request.type()),
            request.dueDate(),
            request.categoryId(),
            request.bankAccountId());

    var result = createUseCase.execute(userId, input);

    if (result.isFailure()) {
      return badRequest(result);
    }

    return ResponseEntity.status(HttpStatus.CREATED).body(TransactionResponse.from(result.value()));
  }

  @GetMapping
  ResponseEntity<List<TransactionResponse>> list() {
    var userId = currentIdentity.findCurrentAuthenticadedUser();
    var result = listUseCase.execute(userId);

    var responses = result.value().stream().map(TransactionResponse::from).toList();

    return ResponseEntity.ok(responses);
  }

  @GetMapping("/{id}")
  ResponseEntity<Object> find(@PathVariable UUID id) {
    var userId = currentIdentity.findCurrentAuthenticadedUser();
    var result = findUseCase.execute(userId, id);

    if (result.isFailure()) {
      return notFound(result);
    }

    return ResponseEntity.ok(TransactionResponse.from(result.value()));
  }

  @PutMapping("/{id}")
  ResponseEntity<Object> update(
      @PathVariable UUID id, @Valid @RequestBody UpdateTransactionRequest request) {
    var userId = currentIdentity.findCurrentAuthenticadedUser();
    var input =
        new UpdateTransactionInput(
            request.description(),
            request.amount(),
            parseType(request.type()),
            request.dueDate(),
            request.categoryId(),
            request.bankAccountId());

    var result = updateUseCase.execute(userId, id, input);

    if (result.isFailure()) {
      return badRequest(result);
    }

    return ResponseEntity.ok(TransactionResponse.from(result.value()));
  }

  @DeleteMapping("/{id}")
  ResponseEntity<Object> delete(@PathVariable UUID id) {
    var userId = currentIdentity.findCurrentAuthenticadedUser();
    var result = deleteUseCase.execute(userId, id);

    if (result.isFailure()) {
      return notFound(result);
    }

    return ResponseEntity.noContent().build();
  }

  @PostMapping("/bundles")
  ResponseEntity<Object> createBundle(@Valid @RequestBody CreateBundleRequest request) {
    var userId = currentIdentity.findCurrentAuthenticadedUser();
    var input =
        new CreateTransactionBundleInput(
            request.description(),
            request.totalAmount(),
            request.totalInstallments(),
            request.firstDueDate(),
            request.categoryId(),
            request.bankAccountId());

    var result = bundleUseCase.execute(userId, input);

    if (result.isFailure()) {
      return badRequest(result);
    }

    return ResponseEntity.status(HttpStatus.CREATED)
        .body(TransactionBundleResponse.from(result.value()));
  }

  @PostMapping("/check-in")
  ResponseEntity<Object> checkIn(@Valid @RequestBody DailyCheckInRequest request) {
    var userId = currentIdentity.findCurrentAuthenticadedUser();

    var expenses =
        request.untrackedExpenses() != null
            ? request.untrackedExpenses().stream()
                .map(
                    e ->
                        new DailyCheckInInput.UntrackedExpense(
                            e.description(), e.amount(), e.categoryId()))
                .toList()
            : List.<DailyCheckInInput.UntrackedExpense>of();

    var input =
        new DailyCheckInInput(
            request.date(),
            request.liquidBalance(),
            request.targetSavings(),
            request.flexibleBudgetCap(),
            expenses,
            request.confirmedPendingTransactionIds() != null
                ? request.confirmedPendingTransactionIds()
                : List.of());

    var result = checkInUseCase.execute(userId, input);

    if (result.isFailure()) {
      return badRequest(result);
    }

    return ResponseEntity.ok(DailyCheckInResponse.from(result.value()));
  }

  @PostMapping("/simulations")
  ResponseEntity<Object> simulate(@RequestBody SimulationRequest request) {
    var userId = currentIdentity.findCurrentAuthenticadedUser();

    var input =
        new SimulationInput(
            request.liquidBalance(),
            request.targetSavings(),
            request.flexibleBudgetCap(),
            request.date(),
            request.totalAmount(),
            request.installments(),
            request.firstDueDate());

    var result = simulationUseCase.execute(userId, input);

    if (result.isFailure()) {
      return badRequest(result);
    }

    var cycles =
        result.value().cycles().stream()
            .map(
                c ->
                    new SimulationCycleResponse(
                        c.cycleStart(),
                        c.cycleEnd(),
                        c.s2sToday(),
                        c.s2sReduction(),
                        c.s2sReductionPercent(),
                        c.projectedBalance(),
                        c.healthStatus(),
                        c.bottleneck()))
            .toList();

    return ResponseEntity.ok(java.util.Map.of("cycles", cycles));
  }

  private @Nullable TransactionType parseType(String type) {
    try {
      return TransactionType.valueOf(type);
    } catch (IllegalArgumentException | NullPointerException e) {
      return null;
    }
  }

  private ResponseEntity<Object> badRequest(OperationResult<?> result) {
    return ResponseEntity.badRequest().body(new ErrorResponse(result.errors()));
  }

  private ResponseEntity<Object> notFound(OperationResult<?> result) {
    return ResponseEntity.status(HttpStatus.NOT_FOUND).body(new ErrorResponse(result.errors()));
  }
}
