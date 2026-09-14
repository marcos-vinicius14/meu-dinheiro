package com.marcos.meudinheiro.bankaccount;

import static org.assertj.core.api.Assertions.assertThat;

import java.util.UUID;

import com.marcos.meudinheiro.IntegrationTestSupport;
import org.junit.jupiter.api.Test;

class BankAccountIT extends IntegrationTestSupport {

    @Test
    void createBankAccountReturnsCreatedWithCurrentBalanceEqualToInitialBalance() throws Exception {
        var email = "account-create@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        var result = authenticated(session).post("/bankaccounts", """
                {
                  "name": "Minha Conta Corrente",
                  "type": "CHECKING",
                  "initialBalance": 1500.50
                }
                """);

        assertThat(result.status()).isEqualTo(201);
        assertThat(result.body()).contains("\"name\":\"Minha Conta Corrente\"");
        assertThat(result.body()).contains("\"type\":\"CHECKING\"");
        assertThat(result.body()).contains("\"initialBalance\":1500.50");
        assertThat(result.body()).contains("\"currentBalance\":1500.50");
    }

    @Test
    void createBankAccountWithoutInitialBalanceDefaultsToZero() throws Exception {
        var email = "account-default-balance@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        var result = authenticated(session).post("/bankaccounts", """
                {
                  "name": "Carteira",
                  "type": "CASH"
                }
                """);

        assertThat(result.status()).isEqualTo(201);
        assertThat(result.body()).contains("\"initialBalance\":0.00");
        assertThat(result.body()).contains("\"currentBalance\":0.00");
    }

    @Test
    void createBankAccountWithoutNameReturnsBadRequest() throws Exception {
        var email = "account-no-name@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        var result = authenticated(session).post("/bankaccounts", """
                {
                  "type": "CHECKING"
                }
                """);

        assertThat(result.status()).isEqualTo(400);
    }

    @Test
    void createBankAccountWithoutTypeReturnsBadRequest() throws Exception {
        var email = "account-no-type@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        var result = authenticated(session).post("/bankaccounts", """
                {
                  "name": "Conta Sem Tipo"
                }
                """);

        assertThat(result.status()).isEqualTo(400);
    }

    @Test
    void createBankAccountWithInvalidTypeReturnsBadRequest() throws Exception {
        var email = "account-invalid-type@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        var result = authenticated(session).post("/bankaccounts", """
                {
                  "name": "Conta Inválida",
                  "type": "UNKNOWN"
                }
                """);

        assertThat(result.status()).isEqualTo(400);
    }

    @Test
    void createBankAccountExceedingLimitReturnsBadRequest() throws Exception {
        var email = "account-limit@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        createAccount(session, "Conta 1", "CHECKING", "0.00");
        createAccount(session, "Conta 2", "CASH", "0.00");
        createAccount(session, "Conta 3", "INVESTMENT", "0.00");

        var result = authenticated(session).post("/bankaccounts", """
                {
                  "name": "Conta 4",
                  "type": "CHECKING"
                }
                """);

        assertThat(result.status()).isEqualTo(400);
        assertThat(result.body()).contains("Limite de 3 contas atingido");
    }

    @Test
    void listBankAccountsReturnsOnlyOwnedAccounts() throws Exception {
        var ownerEmail = "account-owner@example.com";
        var otherEmail = "account-other@example.com";
        var password = "password123";
        createUser(ownerEmail, password);
        createUser(otherEmail, password);
        var ownerSession = login(ownerEmail, password);
        var otherSession = login(otherEmail, password);

        createAccount(ownerSession, "Conta do Dono", "CHECKING", "100.00");
        createAccount(otherSession, "Conta de Outro", "CHECKING", "200.00");

        var result = authenticated(ownerSession).get("/bankaccounts");

        assertThat(result.status()).isEqualTo(200);
        assertThat(result.body()).contains("Conta do Dono");
        assertThat(result.body()).doesNotContain("Conta de Outro");
    }

    @Test
    void findBankAccountReturnsOwnedAccount() throws Exception {
        var email = "account-find@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        var accountId = createAccount(session, "Conta para Buscar", "CHECKING", "500.00");

        var result = authenticated(session).get("/bankaccounts/" + accountId);

        assertThat(result.status()).isEqualTo(200);
        assertThat(result.body()).contains("\"id\":\"" + accountId + "\"");
        assertThat(result.body()).contains("Conta para Buscar");
    }

    @Test
    void findBankAccountFromAnotherUserReturnsNotFound() throws Exception {
        var ownerEmail = "account-find-owner@example.com";
        var otherEmail = "account-find-other@example.com";
        var password = "password123";
        createUser(ownerEmail, password);
        createUser(otherEmail, password);
        var ownerSession = login(ownerEmail, password);
        var otherSession = login(otherEmail, password);

        var accountId = createAccount(ownerSession, "Conta Privada", "CHECKING", "0.00");

        var result = authenticated(otherSession).get("/bankaccounts/" + accountId);

        assertThat(result.status()).isEqualTo(404);
    }

    @Test
    void updateBankAccountUpdatesNameAndInitialBalance() throws Exception {
        var email = "account-update@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        var accountId = createAccount(session, "Nome Antigo", "CHECKING", "100.00");

        var result = authenticated(session).put("/bankaccounts/" + accountId, """
                {
                  "name": "Nome Novo",
                  "initialBalance": 250.00
                }
                """);

        assertThat(result.status()).isEqualTo(200);
        assertThat(result.body()).contains("\"name\":\"Nome Novo\"");
        assertThat(result.body()).contains("\"initialBalance\":250.00");
        assertThat(result.body()).contains("\"currentBalance\":250.00");
        assertThat(result.body()).contains("\"type\":\"CHECKING\"");
    }

    @Test
    void updateBankAccountFromAnotherUserReturnsNotFound() throws Exception {
        var ownerEmail = "account-update-owner@example.com";
        var otherEmail = "account-update-other@example.com";
        var password = "password123";
        createUser(ownerEmail, password);
        createUser(otherEmail, password);
        var ownerSession = login(ownerEmail, password);
        var otherSession = login(otherEmail, password);

        var accountId = createAccount(ownerSession, "Conta do Dono", "CHECKING", "0.00");

        var result = authenticated(otherSession).put("/bankaccounts/" + accountId, """
                {
                  "name": "Tentativa",
                  "initialBalance": 100.00
                }
                """);

        assertThat(result.status()).isEqualTo(404);
    }

    @Test
    void deleteBankAccountWithoutTransactionsReturnsNoContent() throws Exception {
        var email = "account-delete@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        var accountId = createAccount(session, "Conta para Deletar", "CHECKING", "0.00");

        var result = authenticated(session).delete("/bankaccounts/" + accountId);

        assertThat(result.status()).isEqualTo(204);

        var findResult = authenticated(session).get("/bankaccounts/" + accountId);
        assertThat(findResult.status()).isEqualTo(404);
    }

    @Test
    void deleteBankAccountWithTransactionsReturnsConflict() throws Exception {
        var email = "account-delete-with-tx@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        var accountId = createAccount(session, "Conta com Transação", "CHECKING", "0.00");
        var userId = findUserIdByEmail(email);
        var categoryId = createCategoryViaSql(userId);

        jdbcTemplate.update("""
                INSERT INTO tb_transactions (id, user_id, bank_account_id, category_id, description, value, type, status, due_date)
                VALUES (uuidv7(), ?, ?, ?, 'Teste', 50.00, 'FLEXIBLE_EXPENSE', 'CONFIRMED', CURRENT_DATE)
                """, userId, accountId, categoryId);

        var result = authenticated(session).delete("/bankaccounts/" + accountId);

        assertThat(result.status()).isEqualTo(409);
        assertThat(result.body()).contains("Conta possui transações vinculadas");
    }

    @Test
    void deleteBankAccountFromAnotherUserReturnsNotFound() throws Exception {
        var ownerEmail = "account-delete-owner@example.com";
        var otherEmail = "account-delete-other@example.com";
        var password = "password123";
        createUser(ownerEmail, password);
        createUser(otherEmail, password);
        var ownerSession = login(ownerEmail, password);
        var otherSession = login(otherEmail, password);

        var accountId = createAccount(ownerSession, "Conta do Dono", "CHECKING", "0.00");

        var result = authenticated(otherSession).delete("/bankaccounts/" + accountId);

        assertThat(result.status()).isEqualTo(404);
    }

    @Test
    void unauthenticatedAccessReturnsUnauthorized() {
        var result = restTemplate.getForEntity("/bankaccounts", String.class);

        assertThat(result.getStatusCode().value()).isEqualTo(401);
    }
}
