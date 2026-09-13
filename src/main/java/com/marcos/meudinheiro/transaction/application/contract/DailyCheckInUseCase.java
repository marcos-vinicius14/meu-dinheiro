package com.marcos.meudinheiro.transaction.application.contract;

import com.marcos.meudinheiro.transaction.application.contract.dto.DailyCheckInInput;
import com.marcos.meudinheiro.transaction.application.contract.dto.DailyCheckInOutput;
import com.marcos.meudinheiro.shared.notification.OperationResult;

import java.util.UUID;

public interface DailyCheckInUseCase {
    OperationResult<DailyCheckInOutput> execute(UUID userId, DailyCheckInInput input);
}
