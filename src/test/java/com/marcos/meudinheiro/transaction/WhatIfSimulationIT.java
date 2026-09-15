package com.marcos.meudinheiro.transaction;

import static org.assertj.core.api.Assertions.assertThat;

import com.marcos.meudinheiro.IntegrationTestSupport;
import org.junit.jupiter.api.Test;

class WhatIfSimulationIT extends IntegrationTestSupport {

  @Test
  void simulatePurchaseReturnsProjectionForCurrentCycle() throws Exception {
    var email = "whatif-single@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);

    var result =
        authenticated(session)
            .post(
                "/transactions/simulations",
                """
                {
                  "liquidBalance": 1000.00,
                  "targetSavings": 100.00,
                  "flexibleBudgetCap": 900.00,
                  "date": "2026-09-13",
                  "totalAmount": 300.00,
                  "installments": 1,
                  "firstDueDate": "2026-09-13"
                }
                """);

    assertThat(result.status()).isEqualTo(200);
    assertThat(result.body()).contains("\"cycles\":");
    assertThat(result.body()).contains("\"s2sToday\":");
    assertThat(result.body()).contains("\"healthStatus\":");
    assertThat(result.body()).contains("\"bottleneck\":");
  }

  @Test
  void simulateInstallmentPurchaseProjectsAllCycles() throws Exception {
    var email = "whatif-installments@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);

    var result =
        authenticated(session)
            .post(
                "/transactions/simulations",
                """
                {
                  "liquidBalance": 1000.00,
                  "targetSavings": 100.00,
                  "flexibleBudgetCap": 900.00,
                  "date": "2026-09-13",
                  "totalAmount": 600.00,
                  "installments": 6,
                  "firstDueDate": "2026-09-13"
                }
                """);

    assertThat(result.status()).isEqualTo(200);
    assertThat(result.body()).contains("\"cycleStart\":\"2026-09-01\"");
    assertThat(result.body()).contains("\"cycleStart\":\"2026-10-01\"");
  }

  @Test
  void simulateWithInvalidInstallmentsReturnsBadRequest() throws Exception {
    var email = "whatif-invalid@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);

    var result =
        authenticated(session)
            .post(
                "/transactions/simulations",
                """
                {
                  "liquidBalance": 1000.00,
                  "targetSavings": 0,
                  "flexibleBudgetCap": 900.00,
                  "date": "2026-09-13",
                  "totalAmount": 300.00,
                  "installments": 0,
                  "firstDueDate": "2026-09-13"
                }
                """);

    assertThat(result.status()).isEqualTo(400);
  }

  @Test
  void unauthenticatedSimulationReturnsUnauthorized() {
    var result = restTemplate.postForEntity("/transactions/simulations", null, String.class);

    assertThat(result.getStatusCode().value()).isEqualTo(401);
  }
}
