package com.marcos.meudinheiro.transaction;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import com.marcos.meudinheiro.identity.AuthenticationTestSupport;
import org.junit.jupiter.api.Test;
import org.springframework.http.MediaType;

class WhatIfSimulationIT extends AuthenticationTestSupport {

    @Test
    void simulatePurchaseReturnsProjectionForCurrentCycle() throws Exception {
        var email = "whatif-single@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        mockMvc.perform(post("/transactions/simulations")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "liquidBalance": 1000.00,
                                  "targetSavings": 100.00,
                                  "flexibleBudgetCap": 900.00,
                                  "date": "2026-09-13",
                                  "totalAmount": 300.00,
                                  "installments": 1,
                                  "firstDueDate": "2026-09-13"
                                }
                                """))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.cycles.length()").value(1))
                .andExpect(jsonPath("$.cycles[0].s2sToday").exists())
                .andExpect(jsonPath("$.cycles[0].healthStatus").exists())
                .andExpect(jsonPath("$.cycles[0].bottleneck").exists());
    }

    @Test
    void simulateInstallmentPurchaseProjectsAllCycles() throws Exception {
        var email = "whatif-installments@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        mockMvc.perform(post("/transactions/simulations")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "liquidBalance": 1000.00,
                                  "targetSavings": 100.00,
                                  "flexibleBudgetCap": 900.00,
                                  "date": "2026-09-13",
                                  "totalAmount": 600.00,
                                  "installments": 6,
                                  "firstDueDate": "2026-09-13"
                                }
                                """))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.cycles.length()").value(6))
                .andExpect(jsonPath("$.cycles[0].cycleStart").value("2026-09-01"))
                .andExpect(jsonPath("$.cycles[1].cycleStart").value("2026-10-01"));
    }

    @Test
    void simulateWithInvalidInstallmentsReturnsBadRequest() throws Exception {
        var email = "whatif-invalid@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        mockMvc.perform(post("/transactions/simulations")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "liquidBalance": 1000.00,
                                  "targetSavings": 0,
                                  "flexibleBudgetCap": 900.00,
                                  "date": "2026-09-13",
                                  "totalAmount": 300.00,
                                  "installments": 0,
                                  "firstDueDate": "2026-09-13"
                                }
                                """))
                .andExpect(status().isBadRequest());
    }

    @Test
    void unauthenticatedSimulationReturnsUnauthorized() throws Exception {
        mockMvc.perform(post("/transactions/simulations")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{}"))
                .andExpect(status().isUnauthorized());
    }

}
