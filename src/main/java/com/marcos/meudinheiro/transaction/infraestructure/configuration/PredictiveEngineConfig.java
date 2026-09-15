package com.marcos.meudinheiro.transaction.infraestructure.configuration;

import com.marcos.meudinheiro.transaction.domain.service.PredictiveEngine;
import com.marcos.meudinheiro.transaction.domain.service.WhatIfSimulator;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class PredictiveEngineConfig {

  @Bean
  public PredictiveEngine predictiveEngine() {
    return new PredictiveEngine();
  }

  @Bean
  public WhatIfSimulator whatIfSimulator(PredictiveEngine predictiveEngine) {
    return new WhatIfSimulator(predictiveEngine);
  }
}
