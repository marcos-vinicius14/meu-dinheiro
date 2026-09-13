package com.marcos.meudinheiro.transaction;

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

class TransactionIT extends AuthenticationTestSupport {

    @Test
    void createProjectedTransactionReturnsCreated() throws Exception {
        var email = "tx-create@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);
        var categoryId = createCategoryViaApi(session, "Alimentação", true);

        mockMvc.perform(post("/transactions")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "Supermercado",
                                  "amount": 150.50,
                                  "type": "FLEXIBLE_EXPENSE",
                                  "dueDate": "2026-09-20",
                                  "categoryId": "%s"
                                }
                                """.formatted(categoryId)))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.id").exists())
                .andExpect(jsonPath("$.description").value("Supermercado"))
                .andExpect(jsonPath("$.amount").value(150.50))
                .andExpect(jsonPath("$.type").value("FLEXIBLE_EXPENSE"))
                .andExpect(jsonPath("$.status").value("PROJECTED"))
                .andExpect(jsonPath("$.dueDate").value("2026-09-20"))
                .andExpect(jsonPath("$.paymentDate").doesNotExist());
    }

    @Test
    void createTransactionWithBankAccountReturnsCreated() throws Exception {
        var email = "tx-create-account@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);
        var categoryId = createCategoryViaApi(session, "Moradia", false);
        var accountId = createBankAccountViaApi(session, "Conta Corrente");

        mockMvc.perform(post("/transactions")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "Aluguel",
                                  "amount": 1200.00,
                                  "type": "FIXED_EXPENSE",
                                  "dueDate": "2026-09-10",
                                  "categoryId": "%s",
                                  "bankAccountId": "%s"
                                }
                                """.formatted(categoryId, accountId)))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.bankAccountId").value(accountId.toString()));
    }

    @Test
    void createTransactionWithoutCategoryReturnsBadRequest() throws Exception {
        var email = "tx-no-category@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        mockMvc.perform(post("/transactions")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "Sem categoria",
                                  "amount": 100.00,
                                  "type": "FLEXIBLE_EXPENSE",
                                  "dueDate": "2026-09-20"
                                }
                                """))
                .andExpect(status().isBadRequest());
    }

    @Test
    void createTransactionWithZeroAmountReturnsBadRequest() throws Exception {
        var email = "tx-zero@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);
        var categoryId = createCategoryViaApi(session, "Zero", true);

        mockMvc.perform(post("/transactions")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "Zero",
                                  "amount": 0,
                                  "type": "FLEXIBLE_EXPENSE",
                                  "dueDate": "2026-09-20",
                                  "categoryId": "%s"
                                }
                                """.formatted(categoryId)))
                .andExpect(status().isBadRequest());
    }

    @Test
    void createTransactionWithInstallmentTypeReturnsBadRequest() throws Exception {
        var email = "tx-installment@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);
        var categoryId = createCategoryViaApi(session, "Parcela", false);

        mockMvc.perform(post("/transactions")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "Parcela avulsa",
                                  "amount": 100.00,
                                  "type": "INSTALLMENT_EXPENSE",
                                  "dueDate": "2026-09-20",
                                  "categoryId": "%s"
                                }
                                """.formatted(categoryId)))
                .andExpect(status().isBadRequest());
    }

    @Test
    void createTransactionFromAnotherUsersCategoryReturnsBadRequest() throws Exception {
        var ownerEmail = "tx-owner@example.com";
        var otherEmail = "tx-other@example.com";
        var password = "password123";
        createUser(ownerEmail, password);
        createUser(otherEmail, password);
        var ownerSession = login(ownerEmail, password);
        var otherSession = login(otherEmail, password);
        var categoryId = createCategoryViaApi(ownerSession, "Privada", true);

        mockMvc.perform(post("/transactions")
                        .cookie(otherSession.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "Categoria alheia",
                                  "amount": 100.00,
                                  "type": "FLEXIBLE_EXPENSE",
                                  "dueDate": "2026-09-20",
                                  "categoryId": "%s"
                                }
                                """.formatted(categoryId)))
                .andExpect(status().isBadRequest());
    }

    @Test
    void listTransactionsReturnsOnlyOwned() throws Exception {
        var ownerEmail = "tx-list-owner@example.com";
        var otherEmail = "tx-list-other@example.com";
        var password = "password123";
        createUser(ownerEmail, password);
        createUser(otherEmail, password);
        var ownerSession = login(ownerEmail, password);
        var otherSession = login(otherEmail, password);
        var categoryId = createCategoryViaApi(ownerSession, "Lista", true);
        var otherCategoryId = createCategoryViaApi(otherSession, "Lista Alheia", true);

        createTransaction(ownerSession, "Do dono", "100.00", "FLEXIBLE_EXPENSE", categoryId);
        createTransaction(otherSession, "De outro", "200.00", "FLEXIBLE_EXPENSE", otherCategoryId);

        mockMvc.perform(get("/transactions")
                        .cookie(ownerSession.accessTokenCookie()))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.length()").value(1))
                .andExpect(jsonPath("$[0].description").value("Do dono"));
    }

    @Test
    void findTransactionReturnsOwned() throws Exception {
        var email = "tx-find@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);
        var categoryId = createCategoryViaApi(session, "Busca", true);

        var transactionId = createTransaction(session, "Para buscar", "75.00", "FLEXIBLE_EXPENSE", categoryId);

        mockMvc.perform(get("/transactions/{id}", transactionId)
                        .cookie(session.accessTokenCookie()))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.id").value(transactionId.toString()))
                .andExpect(jsonPath("$.description").value("Para buscar"));
    }

    @Test
    void findTransactionFromAnotherUserReturnsNotFound() throws Exception {
        var ownerEmail = "tx-find-owner@example.com";
        var otherEmail = "tx-find-other@example.com";
        var password = "password123";
        createUser(ownerEmail, password);
        createUser(otherEmail, password);
        var ownerSession = login(ownerEmail, password);
        var otherSession = login(otherEmail, password);
        var categoryId = createCategoryViaApi(ownerSession, "Privada", true);

        var transactionId = createTransaction(ownerSession, "Privada", "10.00", "FLEXIBLE_EXPENSE", categoryId);

        mockMvc.perform(get("/transactions/{id}", transactionId)
                        .cookie(otherSession.accessTokenCookie()))
                .andExpect(status().isNotFound());
    }

    @Test
    void updateTransactionChangesFields() throws Exception {
        var email = "tx-update@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);
        var categoryId = createCategoryViaApi(session, "Update", true);

        var transactionId = createTransaction(session, "Antigo", "100.00", "FLEXIBLE_EXPENSE", categoryId);

        mockMvc.perform(put("/transactions/{id}", transactionId)
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "Novo",
                                  "amount": 250.00,
                                  "type": "FIXED_EXPENSE",
                                  "dueDate": "2026-09-25",
                                  "categoryId": "%s"
                                }
                                """.formatted(categoryId)))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.description").value("Novo"))
                .andExpect(jsonPath("$.amount").value(250.00))
                .andExpect(jsonPath("$.type").value("FIXED_EXPENSE"))
                .andExpect(jsonPath("$.dueDate").value("2026-09-25"));
    }

    @Test
    void deleteTransactionReturnsNoContent() throws Exception {
        var email = "tx-delete@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);
        var categoryId = createCategoryViaApi(session, "Delete", true);

        var transactionId = createTransaction(session, "Para deletar", "10.00", "FLEXIBLE_EXPENSE", categoryId);

        mockMvc.perform(delete("/transactions/{id}", transactionId)
                        .cookie(session.accessTokenCookie()))
                .andExpect(status().isNoContent());

        mockMvc.perform(get("/transactions/{id}", transactionId)
                        .cookie(session.accessTokenCookie()))
                .andExpect(status().isNotFound());
    }

    @Test
    void createBundleGeneratesCommittedInstallments() throws Exception {
        var email = "tx-bundle@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);
        var categoryId = createCategoryViaApi(session, "Parcelado", false);

        mockMvc.perform(post("/transactions/bundles")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "Notebook",
                                  "totalAmount": 1000.00,
                                  "totalInstallments": 10,
                                  "firstDueDate": "2026-09-15",
                                  "categoryId": "%s"
                                }
                                """.formatted(categoryId)))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.id").exists())
                .andExpect(jsonPath("$.description").value("Notebook"))
                .andExpect(jsonPath("$.totalInstallments").value(10));

        var bundleTransactions = jdbcTemplate.queryForObject(
                "SELECT COUNT(*) FROM tb_transactions WHERE bundle_id IS NOT NULL AND user_id = ?",
                Integer.class,
                findUserIdByEmail(email)
        );

        org.assertj.core.api.Assertions.assertThat(bundleTransactions).isEqualTo(10);
    }

    @Test
    void unauthenticatedAccessReturnsUnauthorized() throws Exception {
        mockMvc.perform(get("/transactions"))
                .andExpect(status().isUnauthorized());
    }

    private UUID createCategoryViaApi(Session session, String description, boolean flexible) throws Exception {
        var result = mockMvc.perform(post("/categories")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "%s",
                                  "icon": "tag",
                                  "isFlexible": %s
                                }
                                """.formatted(description, flexible)))
                .andExpect(status().isCreated())
                .andReturn();

        var response = result.getResponse().getContentAsString();
        var idStart = response.indexOf("\"id\":\"") + 7;
        var idEnd = response.indexOf("\"", idStart);
        return UUID.fromString(response.substring(idStart, idEnd));
    }

    private UUID createBankAccountViaApi(Session session, String name) throws Exception {
        var result = mockMvc.perform(post("/bankaccounts")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "name": "%s",
                                  "type": "CHECKING"
                                }
                                """.formatted(name)))
                .andExpect(status().isCreated())
                .andReturn();

        var response = result.getResponse().getContentAsString();
        var idStart = response.indexOf("\"id\":\"") + 7;
        var idEnd = response.indexOf("\"", idStart);
        return UUID.fromString(response.substring(idStart, idEnd));
    }

    private UUID createTransaction(
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
                                  "dueDate": "2026-09-20",
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

    private UUID findUserIdByEmail(String email) {
        return jdbcTemplate.queryForObject(
                "SELECT id FROM tb_users WHERE email = ?",
                UUID.class,
                email
        );
    }
}
