package com.marcos.meudinheiro.transaction;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import java.util.UUID;

import com.marcos.meudinheiro.identity.AuthenticationTestSupport;
import org.junit.jupiter.api.Test;
import org.springframework.http.MediaType;

class DailyCheckInIT extends AuthenticationTestSupport {

    @Test
    void checkInPersistsUntrackedExpensesAndReturnsRecalibratedS2S() throws Exception {
        var email = "checkin-create@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);
        var categoryId = createFlexibleCategory(session);

        mockMvc.perform(post("/transactions/check-in")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "date": "2026-09-13",
                                  "liquidBalance": 1000.00,
                                  "targetSavings": 100.00,
                                  "flexibleBudgetCap": 900.00,
                                  "untrackedExpenses": [
                                    {
                                      "description": "Café",
                                      "amount": 25.50,
                                      "categoryId": "%s"
                                    }
                                  ],
                                  "confirmedPendingTransactionIds": []
                                }
                                """.formatted(categoryId)))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.s2sCalculated").exists())
                .andExpect(jsonPath("$.healthStatus").exists())
                .andExpect(jsonPath("$.spentToday").value(25.50));

        var persisted = jdbcTemplate.queryForObject(
                "SELECT COUNT(*) FROM tb_transactions WHERE description = 'Café' AND status = 'CONFIRMED'",
                Integer.class
        );
        org.assertj.core.api.Assertions.assertThat(persisted).isEqualTo(1);
    }

    @Test
    void checkInConfirmsSelectedPendingTransactions() throws Exception {
        var email = "checkin-confirm@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);
        var categoryId = createFlexibleCategory(session);

        var transactionId = createPendingTransaction(session, "Conta pendente", "200.00", categoryId);

        mockMvc.perform(post("/transactions/check-in")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "date": "2026-09-13",
                                  "liquidBalance": 1000.00,
                                  "targetSavings": 0,
                                  "flexibleBudgetCap": 900.00,
                                  "untrackedExpenses": [],
                                  "confirmedPendingTransactionIds": ["%s"]
                                }
                                """.formatted(transactionId)))
                .andExpect(status().isOk());

        var status = jdbcTemplate.queryForObject(
                "SELECT status FROM tb_transactions WHERE id = ?",
                String.class,
                transactionId
        );
        org.assertj.core.api.Assertions.assertThat(status).isEqualTo("CONFIRMED");
    }

    @Test
    void checkInPersistsSnapshotIdempotently() throws Exception {
        var email = "checkin-snapshot@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);
        var categoryId = createFlexibleCategory(session);

        var body = """
                {
                  "date": "2026-09-13",
                  "liquidBalance": 1000.00,
                  "targetSavings": 0,
                  "flexibleBudgetCap": 900.00,
                  "untrackedExpenses": [],
                  "confirmedPendingTransactionIds": []
                }
                """;

        mockMvc.perform(post("/transactions/check-in")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(body))
                .andExpect(status().isOk());

        mockMvc.perform(post("/transactions/check-in")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(body))
                .andExpect(status().isOk());

        var snapshots = jdbcTemplate.queryForObject(
                "SELECT COUNT(*) FROM tb_check_in_snapshots s JOIN tb_users u ON u.id = s.user_id WHERE u.email = ? AND s.check_in_date = DATE '2026-09-13'",
                Integer.class,
                email
        );
        org.assertj.core.api.Assertions.assertThat(snapshots).isEqualTo(1);
    }

    @Test
    void checkInWithLowLiquidityReturnsRestricted() throws Exception {
        var email = "checkin-restricted@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);
        var categoryId = createFlexibleCategory(session);

        mockMvc.perform(post("/transactions/check-in")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "date": "2026-09-13",
                                  "liquidBalance": 10.00,
                                  "targetSavings": 0,
                                  "flexibleBudgetCap": 900.00,
                                  "untrackedExpenses": [],
                                  "confirmedPendingTransactionIds": []
                                }
                                """))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.healthStatus").value("RESTRICTED"));
    }

    @Test
    void checkInWithCommitmentsExceedingLiquidityReturnsDeficitRisk() throws Exception {
        var email = "checkin-deficit-commit@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);
        var categoryId = createFlexibleCategory(session);

        mockMvc.perform(post("/transactions/bundles")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "Cartão",
                                  "totalAmount": 6000.00,
                                  "totalInstallments": 12,
                                  "firstDueDate": "2026-09-13",
                                  "categoryId": "%s"
                                }
                                """.formatted(categoryId)))
                .andExpect(status().isCreated());

        mockMvc.perform(post("/transactions/check-in")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "date": "2026-09-13",
                                  "liquidBalance": 10.00,
                                  "targetSavings": 0,
                                  "flexibleBudgetCap": 900.00,
                                  "untrackedExpenses": [],
                                  "confirmedPendingTransactionIds": []
                                }
                                """))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.healthStatus").value("DEFICIT_RISK"))
                .andExpect(jsonPath("$.s2sCalculated").value(0.00));
    }

    @Test
    void unauthenticatedCheckInReturnsUnauthorized() throws Exception {
        mockMvc.perform(post("/transactions/check-in")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{}"))
                .andExpect(status().isUnauthorized());
    }

    private UUID createFlexibleCategory(Session session) throws Exception {
        var result = mockMvc.perform(post("/categories")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "Flexível",
                                  "icon": "tag",
                                  "isFlexible": true
                                }
                                """))
                .andExpect(status().isCreated())
                .andReturn();

        var response = result.getResponse().getContentAsString();
        var idStart = response.indexOf("\"id\":\"") + 7;
        var idEnd = response.indexOf("\"", idStart);
        return UUID.fromString(response.substring(idStart, idEnd));
    }

    private UUID createTransactionWithStatus(
            Session session,
            String description,
            String amount,
            String type,
            UUID categoryId
    ) throws Exception {
        var result = mockMvc.perform(post("/transactions")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "%s",
                                  "amount": %s,
                                  "type": "%s",
                                  "dueDate": "2026-09-13",
                                  "categoryId": "%s"
                                }
                                """.formatted(description, amount, type, categoryId)))
                .andExpect(status().isCreated())
                .andReturn();

        var response = result.getResponse().getContentAsString();
        var idStart = response.indexOf("\"id\":\"") + 7;
        var idEnd = response.indexOf("\"", idStart);
        return UUID.fromString(response.substring(idStart, idEnd));
    }

    private UUID createPendingTransaction(
            Session session,
            String description,
            String amount,
            UUID categoryId
    ) throws Exception {
        var result = mockMvc.perform(post("/transactions")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "%s",
                                  "amount": %s,
                                  "type": "FLEXIBLE_EXPENSE",
                                  "dueDate": "2026-09-13",
                                  "categoryId": "%s"
                                }
                                """.formatted(description, amount, categoryId)))
                .andExpect(status().isCreated())
                .andReturn();

        var response = result.getResponse().getContentAsString();
        var idStart = response.indexOf("\"id\":\"") + 7;
        var idEnd = response.indexOf("\"", idStart);
        return UUID.fromString(response.substring(idStart, idEnd));
    }
}
