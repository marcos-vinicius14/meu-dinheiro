package com.marcos.meudinheiro.transaction.application.usecase;

import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.shared.valueobjects.Money;
import com.marcos.meudinheiro.transaction.application.contract.DailyCheckInUseCase;
import com.marcos.meudinheiro.transaction.application.contract.TransactionCategoryResolver;
import com.marcos.meudinheiro.transaction.application.contract.TransactionUserResolver;
import com.marcos.meudinheiro.transaction.application.contract.constant.TransactionMessages;
import com.marcos.meudinheiro.transaction.application.contract.dto.DailyCheckInInput;
import com.marcos.meudinheiro.transaction.application.contract.dto.DailyCheckInOutput;
import com.marcos.meudinheiro.transaction.domain.model.CheckInSnapshotModel;
import com.marcos.meudinheiro.transaction.domain.model.TransactionModel;
import com.marcos.meudinheiro.transaction.domain.service.CurrentState;
import com.marcos.meudinheiro.transaction.domain.service.CycleContext;
import com.marcos.meudinheiro.transaction.domain.service.EngineResult;
import com.marcos.meudinheiro.transaction.domain.service.PredictiveEngine;
import com.marcos.meudinheiro.transaction.domain.service.TransactionSnapshot;
import com.marcos.meudinheiro.transaction.domain.valueobject.DateInterval;
import com.marcos.meudinheiro.transaction.infraestructure.repository.CheckInSnapshotRepository;
import com.marcos.meudinheiro.transaction.infraestructure.repository.TransactionRepository;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.LocalDate;
import java.time.YearMonth;
import java.util.ArrayList;
import java.util.List;
import java.util.UUID;

@Service
public class DailyCheckInUseCaseImpl implements DailyCheckInUseCase {

    private final TransactionRepository transactionRepository;
    private final CheckInSnapshotRepository snapshotRepository;
    private final TransactionUserResolver userResolver;
    private final TransactionCategoryResolver categoryResolver;
    private final PredictiveEngine engine;

    public DailyCheckInUseCaseImpl(
            TransactionRepository transactionRepository,
            CheckInSnapshotRepository snapshotRepository,
            TransactionUserResolver userResolver,
            TransactionCategoryResolver categoryResolver,
            PredictiveEngine engine
    ) {
        this.transactionRepository = transactionRepository;
        this.snapshotRepository = snapshotRepository;
        this.userResolver = userResolver;
        this.categoryResolver = categoryResolver;
        this.engine = engine;
    }

    @Transactional
    @Override
    public OperationResult<DailyCheckInOutput> execute(UUID userId, DailyCheckInInput input) {
        if (input.date() == null) {
            return OperationResult.failure("Data do check-in é obrigatória");
        }

        var user = userResolver.resolve(userId);

        // Validação completa antes de qualquer escrita: failure não pode deixar
        // escritas parciais commitadas (e retry não pode duplicar despesas).
        var newExpenses = new ArrayList<TransactionModel>();
        for (var expense : input.untrackedExpenses()) {
            var categoryResult = categoryResolver.resolve(userId, expense.categoryId());
            if (categoryResult.isFailure()) {
                return OperationResult.failure(categoryResult.errors());
            }

            try {
                var transaction = TransactionModel.create(
                        expense.description(),
                        com.marcos.meudinheiro.transaction.domain.enums.TransactionType.FLEXIBLE_EXPENSE,
                        expense.amount() != null ? new Money(expense.amount()) : null,
                        input.date(),
                        user,
                        null,
                        categoryResult.value()
                );
                transaction.confirm(input.date());
                newExpenses.add(transaction);
            } catch (IllegalArgumentException e) {
                return OperationResult.failure(e.getMessage());
            }
        }

        var pendingTransactions = new ArrayList<TransactionModel>();
        for (var transactionId : input.confirmedPendingTransactionIds()) {
            var transaction = transactionRepository.findById(transactionId)
                    .filter(t -> t.belongsTo(userId))
                    .orElse(null);

            if (transaction == null) {
                return OperationResult.failure(TransactionMessages.TRANSACTION_NOT_FOUND);
            }
            if (transaction.getStatus() == com.marcos.meudinheiro.transaction.domain.enums.TransactionStatus.CANCELED) {
                return OperationResult.failure(
                        "Transação cancelada não pode ser confirmada");
            }

            pendingTransactions.add(transaction);
        }

        // Persistência/mutação só depois de tudo validado
        var spentToday = Money.zero();
        for (var transaction : newExpenses) {
            transactionRepository.save(transaction);
            spentToday = spentToday.add(transaction.getAmount());
        }
        for (var transaction : pendingTransactions) {
            transaction.confirm(input.date());
        }

        var engineResult = calculateEngineResult(userId, input, spentToday);

        var delta = engineResult.s2sToday().subtract(spentToday);

        upsertSnapshot(user, input.date(), engineResult, spentToday, delta);

        var nextDayS2S = nextDayS2S(userId, input);
        var projectedFreeBalance = engineResult.netLiquidityBeforeFlex().subtract(
                Money.max(
                        Money.max(
                                input.flexibleBudgetCap() != null ? new Money(input.flexibleBudgetCap()) : Money.zero(),
                                Money.zero()
                        ).subtract(engineResult.flexibleSpent()),
                        Money.zero()
                )
        );

        return OperationResult.success(new DailyCheckInOutput(
                engineResult.s2sToday().getValue(),
                spentToday.getValue(),
                delta.getValue(),
                engineResult.healthStatus(),
                nextDayS2S.getValue(),
                projectedFreeBalance.getValue(),
                engineResult.daysRemaining()
        ));
    }

    private EngineResult calculateEngineResult(UUID userId, DailyCheckInInput input, Money spentToday) {
        var context = buildContext(input, input.date());
        return engine.calculate(context, new CurrentState(liquidBalance(input), loadSnapshots(userId)));
    }

    /**
     * S2S recalibrado para o dia seguinte: mesmo estado, avançando um dia no ciclo.
     */
    private Money nextDayS2S(UUID userId, DailyCheckInInput input) {
        var tomorrow = input.date().plusDays(1);
        var context = buildContext(input, tomorrow);
        return engine.calculate(context, new CurrentState(liquidBalance(input), loadSnapshots(userId)))
                .s2sToday();
    }

    private CycleContext buildContext(DailyCheckInInput input, LocalDate date) {
        var cycleInterval = new DateInterval(
                YearMonth.from(date).atDay(1),
                YearMonth.from(date).atEndOfMonth()
        );
        return new CycleContext(
                cycleInterval,
                date,
                input.targetSavings() != null ? new Money(input.targetSavings()) : Money.zero(),
                input.flexibleBudgetCap() != null ? new Money(input.flexibleBudgetCap()) : Money.zero()
        );
    }

    private Money liquidBalance(DailyCheckInInput input) {
        return input.liquidBalance() != null ? new Money(input.liquidBalance()) : Money.zero();
    }

    private List<TransactionSnapshot> loadSnapshots(UUID userId) {
        return transactionRepository.findAllByUserIdOrderByDueDateDesc(userId)
                .stream()
                .map(t -> new TransactionSnapshot(t.getType(), t.getStatus(), t.getAmount(), t.getDueDate()))
                .toList();
    }

    private void upsertSnapshot(
            com.marcos.meudinheiro.user.domain.model.UserModel user,
            LocalDate date,
            EngineResult engineResult,
            Money spentToday,
            Money delta
    ) {
        var existing = snapshotRepository.findByUserIdAndCheckInDate(user.getId(), date);

        if (existing.isPresent()) {
            var snapshot = existing.get();
            snapshot.update(engineResult.s2sToday(), spentToday, delta, engineResult.healthStatus());
            snapshotRepository.save(snapshot);
            return;
        }

        snapshotRepository.save(new CheckInSnapshotModel(
                user,
                date,
                engineResult.s2sToday(),
                spentToday,
                delta,
                engineResult.healthStatus()
        ));
    }
}
