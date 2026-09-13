package com.marcos.meudinheiro.category;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.delete;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import java.util.UUID;

import com.marcos.meudinheiro.identity.AuthenticationTestSupport;
import org.junit.jupiter.api.Test;
import org.springframework.http.MediaType;

class CategoryIT extends AuthenticationTestSupport {

    @Test
    void createCategoryWithIconReturnsCreated() throws Exception {
        var email = "category-create@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        mockMvc.perform(post("/categories")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "Alimentação",
                                  "icon": "food"
                                }
                                """))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.id").exists())
                .andExpect(jsonPath("$.description").value("Alimentação"))
                .andExpect(jsonPath("$.icon").value("food"));
    }

    @Test
    void createCategoryWithoutIconReturnsCreatedWithNullIcon() throws Exception {
        var email = "category-create-no-icon@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        mockMvc.perform(post("/categories")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "Transporte"
                                }
                                """))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.description").value("Transporte"))
                .andExpect(jsonPath("$.icon").doesNotExist());
    }

    @Test
    void createCategoryWithoutDescriptionReturnsBadRequest() throws Exception {
        var email = "category-no-description@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        mockMvc.perform(post("/categories")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "icon": "food"
                                }
                                """))
                .andExpect(status().isBadRequest());
    }

    @Test
    void createCategoryWithBlankDescriptionReturnsBadRequest() throws Exception {
        var email = "category-blank-description@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        mockMvc.perform(post("/categories")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "   "
                                }
                                """))
                .andExpect(status().isBadRequest());
    }

    @Test
    void createCategoryWithTooShortDescriptionReturnsBadRequest() throws Exception {
        var email = "category-short-description@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        mockMvc.perform(post("/categories")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "Ab"
                                }
                                """))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.errors[0]").value("Descrição da categoria deve ter entre 3 e 100 caracteres"));
    }

    @Test
    void createCategoryWithTooLongDescriptionReturnsBadRequest() throws Exception {
        var email = "category-long-description@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        mockMvc.perform(post("/categories")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "%s"
                                }
                                """.formatted("a".repeat(101))))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.errors[0]").value("Descrição da categoria deve ter entre 3 e 100 caracteres"));
    }

    @Test
    void createDuplicatedCategoryForSameUserReturnsBadRequest() throws Exception {
        var email = "category-duplicated@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        createCategory(session, "Moradia", "home");

        mockMvc.perform(post("/categories")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "   Moradia  ",
                                  "icon": "house"
                                }
                                """))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.errors[0]").value("Você já possui uma categoria com essa descrição"));
    }

    @Test
    void createDuplicatedCategoryForDifferentUsersIsAllowed() throws Exception {
        var ownerEmail = "category-dup-owner@example.com";
        var otherEmail = "category-dup-other@example.com";
        var password = "password123";
        createUser(ownerEmail, password);
        createUser(otherEmail, password);
        var ownerSession = login(ownerEmail, password);
        var otherSession = login(otherEmail, password);

        createCategory(ownerSession, "Saúde", "health");

        mockMvc.perform(post("/categories")
                        .cookie(otherSession.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "Saúde",
                                  "icon": "heart"
                                }
                                """))
                .andExpect(status().isCreated());
    }

    @Test
    void listCategoriesReturnsOnlyOwnedCategories() throws Exception {
        var ownerEmail = "category-owner@example.com";
        var otherEmail = "category-other@example.com";
        var password = "password123";
        createUser(ownerEmail, password);
        createUser(otherEmail, password);
        var ownerSession = login(ownerEmail, password);
        var otherSession = login(otherEmail, password);

        createCategory(ownerSession, "Categoria do Dono", "tag");
        createCategory(otherSession, "Categoria de Outro", "tag");

        mockMvc.perform(get("/categories")
                        .cookie(ownerSession.accessTokenCookie()))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.length()").value(1))
                .andExpect(jsonPath("$[0].description").value("Categoria do Dono"));
    }

    @Test
    void findCategoryReturnsOwnedCategory() throws Exception {
        var email = "category-find@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        var categoryId = createCategory(session, "Lazer", "fun");

        mockMvc.perform(get("/categories/{id}", categoryId)
                        .cookie(session.accessTokenCookie()))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.id").value(categoryId.toString()))
                .andExpect(jsonPath("$.description").value("Lazer"))
                .andExpect(jsonPath("$.icon").value("fun"));
    }

    @Test
    void findCategoryFromAnotherUserReturnsNotFound() throws Exception {
        var ownerEmail = "category-find-owner@example.com";
        var otherEmail = "category-find-other@example.com";
        var password = "password123";
        createUser(ownerEmail, password);
        createUser(otherEmail, password);
        var ownerSession = login(ownerEmail, password);
        var otherSession = login(otherEmail, password);

        var categoryId = createCategory(ownerSession, "Categoria Privada", "lock");

        mockMvc.perform(get("/categories/{id}", categoryId)
                        .cookie(otherSession.accessTokenCookie()))
                .andExpect(status().isNotFound());
    }

    @Test
    void findUnknownCategoryReturnsNotFound() throws Exception {
        var email = "category-find-unknown@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        mockMvc.perform(get("/categories/{id}", UUID.randomUUID())
                        .cookie(session.accessTokenCookie()))
                .andExpect(status().isNotFound());
    }

    @Test
    void updateCategoryUpdatesDescriptionAndIcon() throws Exception {
        var email = "category-update@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        var categoryId = createCategory(session, "Nome Antigo", "old");

        mockMvc.perform(put("/categories/{id}", categoryId)
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "Nome Novo",
                                  "icon": "new"
                                }
                                """))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.id").value(categoryId.toString()))
                .andExpect(jsonPath("$.description").value("Nome Novo"))
                .andExpect(jsonPath("$.icon").value("new"));
    }

    @Test
    void updateCategoryFromAnotherUserReturnsNotFound() throws Exception {
        var ownerEmail = "category-update-owner@example.com";
        var otherEmail = "category-update-other@example.com";
        var password = "password123";
        createUser(ownerEmail, password);
        createUser(otherEmail, password);
        var ownerSession = login(ownerEmail, password);
        var otherSession = login(otherEmail, password);

        var categoryId = createCategory(ownerSession, "Categoria do Dono", "lock");

        mockMvc.perform(put("/categories/{id}", categoryId)
                        .cookie(otherSession.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "Tentativa"
                                }
                                """))
                .andExpect(status().isNotFound());
    }

    @Test
    void updateCategoryToDuplicatedDescriptionReturnsBadRequest() throws Exception {
        var email = "category-update-dup@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        createCategory(session, "Primeira", "one");
        var categoryId = createCategory(session, "Segunda", "two");

        mockMvc.perform(put("/categories/{id}", categoryId)
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "Primeira"
                                }
                                """))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.errors[0]").value("Você já possui uma categoria com essa descrição"));
    }

    @Test
    void updateCategoryWithInvalidDescriptionReturnsBadRequest() throws Exception {
        var email = "category-update-invalid@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        var categoryId = createCategory(session, "Válida", "ok");

        mockMvc.perform(put("/categories/{id}", categoryId)
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "Ab"
                                }
                                """))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.errors[0]").value("Descrição da categoria deve ter entre 3 e 100 caracteres"));
    }

    @Test
    void deleteCategoryWithoutTransactionsReturnsNoContent() throws Exception {
        var email = "category-delete@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        var categoryId = createCategory(session, "Categoria para Deletar", "trash");

        mockMvc.perform(delete("/categories/{id}", categoryId)
                        .cookie(session.accessTokenCookie()))
                .andExpect(status().isNoContent());

        mockMvc.perform(get("/categories/{id}", categoryId)
                        .cookie(session.accessTokenCookie()))
                .andExpect(status().isNotFound());
    }

    @Test
    void deleteCategoryWithTransactionsReturnsConflict() throws Exception {
        var email = "category-delete-with-tx@example.com";
        var password = "password123";
        createUser(email, password);
        var session = login(email, password);

        var categoryId = createCategory(session, "Categoria com Transação", "money");
        var userId = findUserIdByEmail(email);
        var accountId = createBankAccount(userId);

        jdbcTemplate.update("""
                INSERT INTO tb_transactions (id, user_id, bank_account_id, category_id, description, value, type, date)
                VALUES (uuidv7(), ?, ?, ?, 'Teste', 50.00, 'EXPENSE', CURRENT_TIMESTAMP)
                """, userId, accountId, categoryId);

        mockMvc.perform(delete("/categories/{id}", categoryId)
                        .cookie(session.accessTokenCookie()))
                .andExpect(status().isConflict())
                .andExpect(jsonPath("$.errors[0]").value("Categoria possui transações vinculadas"));
    }

    @Test
    void deleteCategoryFromAnotherUserReturnsNotFound() throws Exception {
        var ownerEmail = "category-delete-owner@example.com";
        var otherEmail = "category-delete-other@example.com";
        var password = "password123";
        createUser(ownerEmail, password);
        createUser(otherEmail, password);
        var ownerSession = login(ownerEmail, password);
        var otherSession = login(otherEmail, password);

        var categoryId = createCategory(ownerSession, "Categoria do Dono", "lock");

        mockMvc.perform(delete("/categories/{id}", categoryId)
                        .cookie(otherSession.accessTokenCookie()))
                .andExpect(status().isNotFound());
    }

    @Test
    void unauthenticatedAccessReturnsUnauthorized() throws Exception {
        mockMvc.perform(get("/categories"))
                .andExpect(status().isUnauthorized());
    }

    private UUID createCategory(Session session, String description, String icon) throws Exception {
        var result = mockMvc.perform(post("/categories")
                        .cookie(session.accessTokenCookie())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "description": "%s",
                                  "icon": "%s"
                                }
                                """.formatted(description, icon)))
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

    private UUID createBankAccount(UUID userId) {
        return jdbcTemplate.queryForObject(
                "INSERT INTO tb_bank_accounts (id, user_id, name, initial_balance, type) VALUES (uuidv7(), ?, 'Conta Teste', 0, 'CHECKING') RETURNING id",
                UUID.class,
                userId
        );
    }
}
