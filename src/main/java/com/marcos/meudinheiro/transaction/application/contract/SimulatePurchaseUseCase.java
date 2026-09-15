package com.marcos.meudinheiro.transaction.application.contract;

import com.marcos.meudinheiro.shared.notification.OperationResult;
import com.marcos.meudinheiro.transaction.application.contract.dto.SimulationInput;
import com.marcos.meudinheiro.transaction.application.contract.dto.SimulationOutput;
import java.util.UUID;

public interface SimulatePurchaseUseCase {
  OperationResult<SimulationOutput> execute(UUID userId, SimulationInput input);
}
