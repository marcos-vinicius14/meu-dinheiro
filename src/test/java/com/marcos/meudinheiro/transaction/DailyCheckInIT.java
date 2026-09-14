package com.marcos.meudinheiro.transaction;

import static org.assertj.core.api.Assertions.assertThat;

import java.util.UUID;

import com.marcos.meudinheiro.IntegrationTestSupport;
import org.junit.jupiter.api.Test;

class DailyCheckInIT extends IntegrationTestSupport {

    @Test
    void checkInPersistsUntrackedExpensesAndReturnsRecalibratedS2S() throws Exception {
        var email = "checkin-create@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);
        var categoryId = createCategory(session, "Flexível", true);

        var result = authenticated(session).post("/transactions/check-in", """
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
                """.formatted(categoryId));

        assertThat(result.status()).isEqualTo(200);
        assertThat(result.body()).contains("\"s2sCalculated\":");
        assertThat(result.body()).contains("\"healthStatus\":");
        assertThat(result.body()).contains("\"spentToday\":25.50");

        var persisted = jdbcTemplate.queryForObject(
                "SELECT COUNT(*) FROM tb_transactions WHERE description = 'Café' AND status = 'CONFIRMED'",
                Integer.class
        );
        assertThat(persisted).isEqualTo(1);
    }

    @Test
    void checkInConfirmsSelectedPendingTransactions() throws Exception {
        var email = "checkin-confirm@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);
        var categoryId = createCategory(session, "Flexível", true);

        var transactionId = createTransaction(session, "Conta pendente", "200.00", "FLEXIBLE_EXPENSE", categoryId);

        var result = authenticated(session).post("/transactions/check-in", """
                {
                  "date": "2026-09-13",
                  "liquidBalance": 1000.00,
                  "targetSavings": 0,
                  "flexibleBudgetCap": 900.00,
                  "untrackedExpenses": [],
                  "confirmedPendingTransactionIds": ["%s"]
                }
                """.formatted(transactionId));

        assertThat(result.status()).isEqualTo(200);

        var status = jdbcTemplate.queryForObject(
                "SELECT status FROM tb_transactions WHERE id = ?",
                String.class,
                transactionId
        );
        assertThat(status).isEqualTo("CONFIRMED");
    }

    @Test
    void checkInPersistsSnapshotIdempotently() throws Exception {
        var email = "checkin-snapshot@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);
        var categoryId = createCategory(session, "Flexível", true);

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

        assertThat(authenticated(session).post("/transactions/check-in", body).status()).isEqualTo(200);
        assertThat(authenticated(session).post("/transactions/check-in", body).status()).isEqualTo(200);

        var snapshots = jdbcTemplate.queryForObject(
                "SELECT COUNT(*) FROM tb_check_in_snapshots s JOIN tb_users u ON u.id = s.user_id WHERE u.email = ? AND s.check_in_date = DATE '2026-09-13'",
                Integer.class,
                email
        );
        assertThat(snapshots).isEqualTo(1);
    }

    @Test
    void checkInWithLowLiquidityReturnsRestricted() throws Exception {
        var email = "checkin-restricted@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);
        var categoryId = createCategory(session, "Flexível", true);

        var result = authenticated(session).post("/transactions/check-in", """
                {
                  "date": "2026-09-13",
                  "liquidBalance": 10.00,
                  "targetSavings": 0,
                  "flexibleBudgetCap": 900.00,
                  "untrackedExpenses": [],
                  "confirmedPendingTransactionIds": []
                }
                """);

        assertThat(result.status()).isEqualTo(200);
        assertThat(result.body()).contains("\"healthStatus\":\"RESTRICTED\"");
    }

    @Test
    void checkInWithCommitmentsExceedingLiquidityReturnsDeficitRisk() throws Exception {
        var email = "checkin-deficit-commit@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);
        var categoryId = createCategory(session, "Flexível", true);

        var bundleResult = authenticated(session).post("/transactions/bundles", """
                {
                  "description": "Cartão",
                  "totalAmount": 6000.00,
                  "totalInstallments": 12,
                  "firstDueDate": "2026-09-13",
                  "categoryId": "%s"
                }
                """.formatted(categoryId));
        assertThat(bundleResult.status()).isEqualTo(201);

        var result = authenticated(session).post("/transactions/check-in", """
                {
                  "date": "2026-09-13",
                  "liquidBalance": 10.00,
                  "targetSavings": 0,
                  "flexibleBudgetCap": 900.00,
                  "untrackedExpenses": [],
                  "confirmedPendingTransactionIds": []
                }
                """);

        assertThat(result.status()).isEqualTo(200);
        assertThat(result.body()).contains("\"healthStatus\":\"DEFICIT_RISK\"");
        assertThat(result.body()).contains("\"s2sCalculated\":0.00");
    }

    @Test
    void unauthenticatedCheckInReturnsUnauthorized() {
        var result = restTemplate.postForEntity("/transactions/check-in", null, String.class);

        assertThat(result.getStatusCode().value()).isEqualTo(401);
    }
}
