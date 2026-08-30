package com.marcos.meudinheiro.bankaccount;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.delete;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import java.math.BigDecimal;
import java.util.UUID;

import com.marcos.meudinheiro.identity.AuthenticationTestSupport;
import org.junit.jupiter.api.Test;
import org.springframework.http.MediaType;

class BankAccountIT extends AuthenticationTestSupport {

    @Test
    void createBankAccountReturnsCreatedWithCurrentBalanceEqualToInitialBalance() throws Exception {
        var email = "account-create@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        mockMvc.perform(post("/bankaccounts")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "name": "Minha Conta Corrente",
                                  "type": "CHECKING",
                                  "initialBalance": 1500.50
                                }
                                """))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.id").exists())
                .andExpect(jsonPath("$.name").value("Minha Conta Corrente"))
                .andExpect(jsonPath("$.type").value("CHECKING"))
                .andExpect(jsonPath("$.initialBalance").value(1500.50))
                .andExpect(jsonPath("$.currentBalance").value(1500.50));
    }

    @Test
    void createBankAccountWithoutInitialBalanceDefaultsToZero() throws Exception {
        var email = "account-default-balance@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        mockMvc.perform(post("/bankaccounts")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "name": "Carteira",
                                  "type": "CASH"
                                }
                                """))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.initialBalance").value(0.00))
                .andExpect(jsonPath("$.currentBalance").value(0.00));
    }

    @Test
    void createBankAccountWithoutNameReturnsBadRequest() throws Exception {
        var email = "account-no-name@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        mockMvc.perform(post("/bankaccounts")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "type": "CHECKING"
                                }
                                """))
                .andExpect(status().isBadRequest());
    }

    @Test
    void createBankAccountWithoutTypeReturnsBadRequest() throws Exception {
        var email = "account-no-type@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        mockMvc.perform(post("/bankaccounts")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "name": "Conta Sem Tipo"
                                }
                                """))
                .andExpect(status().isBadRequest());
    }

    @Test
    void createBankAccountWithInvalidTypeReturnsBadRequest() throws Exception {
        var email = "account-invalid-type@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        mockMvc.perform(post("/bankaccounts")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "name": "Conta Inválida",
                                  "type": "UNKNOWN"
                                }
                                """))
                .andExpect(status().isBadRequest());
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

        mockMvc.perform(post("/bankaccounts")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "name": "Conta 4",
                                  "type": "CHECKING"
                                }
                                """))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.errors[0]").value("Limite de 3 contas atingido"));
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

        mockMvc.perform(get("/bankaccounts")
                        .cookie(ownerSession.accessTokenCookie()))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.length()").value(1))
                .andExpect(jsonPath("$[0].name").value("Conta do Dono"));
    }

    @Test
    void findBankAccountReturnsOwnedAccount() throws Exception {
        var email = "account-find@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        var accountId = createAccount(session, "Conta para Buscar", "CHECKING", "500.00");

        mockMvc.perform(get("/bankaccounts/{id}", accountId)
                        .cookie(session.accessTokenCookie()))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.id").value(accountId.toString()))
                .andExpect(jsonPath("$.name").value("Conta para Buscar"));
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

        mockMvc.perform(get("/bankaccounts/{id}", accountId)
                        .cookie(otherSession.accessTokenCookie()))
                .andExpect(status().isNotFound());
    }

    @Test
    void updateBankAccountUpdatesNameAndInitialBalance() throws Exception {
        var email = "account-update@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        var accountId = createAccount(session, "Nome Antigo", "CHECKING", "100.00");

        mockMvc.perform(put("/bankaccounts/{id}", accountId)
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "name": "Nome Novo",
                                  "initialBalance": 250.00
                                }
                                """))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.id").value(accountId.toString()))
                .andExpect(jsonPath("$.name").value("Nome Novo"))
                .andExpect(jsonPath("$.initialBalance").value(250.00))
                .andExpect(jsonPath("$.currentBalance").value(250.00))
                .andExpect(jsonPath("$.type").value("CHECKING"));
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

        mockMvc.perform(put("/bankaccounts/{id}", accountId)
                        .cookie(otherSession.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "name": "Tentativa",
                                  "initialBalance": 100.00
                                }
                                """))
                .andExpect(status().isNotFound());
    }

    @Test
    void deleteBankAccountWithoutTransactionsReturnsNoContent() throws Exception {
        var email = "account-delete@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        var accountId = createAccount(session, "Conta para Deletar", "CHECKING", "0.00");

        mockMvc.perform(delete("/bankaccounts/{id}", accountId)
                        .cookie(session.accessTokenCookie()))
                .andExpect(status().isNoContent());

        mockMvc.perform(get("/bankaccounts/{id}", accountId)
                        .cookie(session.accessTokenCookie()))
                .andExpect(status().isNotFound());
    }

    @Test
    void deleteBankAccountWithTransactionsReturnsConflict() throws Exception {
        var email = "account-delete-with-tx@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        var accountId = createAccount(session, "Conta com Transação", "CHECKING", "0.00");
        var userId = findUserIdByEmail(email);
        var categoryId = createCategory(userId);

        jdbcTemplate.update("""
                INSERT INTO tb_transactions (id, user_id, bank_account_id, category_id, description, value, type, date)
                VALUES (uuidv7(), ?, ?, ?, 'Teste', 50.00, 'EXPENSE', CURRENT_TIMESTAMP)
                """, userId, accountId, categoryId);

        mockMvc.perform(delete("/bankaccounts/{id}", accountId)
                        .cookie(session.accessTokenCookie()))
                .andExpect(status().isConflict())
                .andExpect(jsonPath("$.errors[0]").value("Conta possui transações vinculadas"));
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

        mockMvc.perform(delete("/bankaccounts/{id}", accountId)
                        .cookie(otherSession.accessTokenCookie()))
                .andExpect(status().isNotFound());
    }

    @Test
    void unauthenticatedAccessReturnsUnauthorized() throws Exception {
        mockMvc.perform(get("/bankaccounts"))
                .andExpect(status().isUnauthorized());
    }

    private UUID createAccount(Session session, String name, String type, String initialBalance) throws Exception {
        var result = mockMvc.perform(post("/bankaccounts")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "name": "%s",
                                  "type": "%s",
                                  "initialBalance": %s
                                }
                                """.formatted(name, type, initialBalance)))
                .andExpect(status().isCreated())
                .andReturn();

        var response = result.getResponse().getContentAsString();
        var idStart = response.indexOf("\"id\":\"") + 7;
        var idEnd = response.indexOf("\"", idStart);
        return UUID.fromString(response.substring(idStart, idEnd));
    }

    private UUID findUserIdByEmail(String email) {
        return jdbcTemplate.queryForObject(
                "SELECT id FROM tb_users WHERE email = ?",
                UUID.class,
                email
        );
    }

    private UUID createCategory(UUID userId) {
        return jdbcTemplate.queryForObject(
                "INSERT INTO tb_categories (id, user_id, description, icon) VALUES (uuidv7(), ?, 'Categoria Teste', 'icon') RETURNING id",
                UUID.class,
                userId
        );
    }
}
