package com.marcos.meudinheiro.category;

import static org.assertj.core.api.Assertions.assertThat;

import com.marcos.meudinheiro.IntegrationTestSupport;
import java.util.UUID;
import org.junit.jupiter.api.Test;

class CategoryIT extends IntegrationTestSupport {

  @Test
  void createCategoryWithIconReturnsCreated() throws Exception {
    var email = "category-create@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);

    var result =
        authenticated(session)
            .post(
                "/categories",
                """
                {
                  "description": "Alimentação",
                  "icon": "food"
                }
                """);

    assertThat(result.status()).isEqualTo(201);
    assertThat(result.body()).contains("\"description\":\"Alimentação\"");
    assertThat(result.body()).contains("\"icon\":\"food\"");
    assertThat(result.body()).contains("\"isFlexible\":false");
  }

  @Test
  void createCategoryWithoutIconReturnsCreatedWithNullIcon() throws Exception {
    var email = "category-create-no-icon@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);

    var result =
        authenticated(session)
            .post(
                "/categories",
                """
                {
                  "description": "Transporte"
                }
                """);

    assertThat(result.status()).isEqualTo(201);
    assertThat(result.body()).contains("\"description\":\"Transporte\"");
    assertThat(result.body()).contains("\"icon\":null");
  }

  @Test
  void createCategoryWithoutDescriptionReturnsBadRequest() throws Exception {
    var email = "category-no-description@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);

    var result =
        authenticated(session)
            .post(
                "/categories",
                """
                {
                  "icon": "food"
                }
                """);

    assertThat(result.status()).isEqualTo(400);
  }

  @Test
  void createCategoryWithBlankDescriptionReturnsBadRequest() throws Exception {
    var email = "category-blank-description@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);

    var result =
        authenticated(session)
            .post(
                "/categories",
                """
                {
                  "description": "   "
                }
                """);

    assertThat(result.status()).isEqualTo(400);
  }

  @Test
  void createCategoryWithTooShortDescriptionReturnsBadRequest() throws Exception {
    var email = "category-short-description@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);

    var result =
        authenticated(session)
            .post(
                "/categories",
                """
                {
                  "description": "Ab"
                }
                """);

    assertThat(result.status()).isEqualTo(400);
    assertThat(result.body()).contains("Descrição da categoria deve ter entre 3 e 100 caracteres");
  }

  @Test
  void createCategoryWithTooLongDescriptionReturnsBadRequest() throws Exception {
    var email = "category-long-description@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);

    var result =
        authenticated(session)
            .post(
                "/categories",
                """
                {
                  "description": "%s"
                }
                """
                    .formatted("a".repeat(101)));

    assertThat(result.status()).isEqualTo(400);
    assertThat(result.body()).contains("Descrição da categoria deve ter entre 3 e 100 caracteres");
  }

  @Test
  void createDuplicatedCategoryForSameUserReturnsBadRequest() throws Exception {
    var email = "category-duplicated@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);

    createCategory(session, "Moradia", false);

    var result =
        authenticated(session)
            .post(
                "/categories",
                """
                {
                  "description": "   Moradia  ",
                  "icon": "house"
                }
                """);

    assertThat(result.status()).isEqualTo(400);
    assertThat(result.body()).contains("Você já possui uma categoria com essa descrição");
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

    createCategory(ownerSession, "Saúde", false);

    var result =
        authenticated(otherSession)
            .post(
                "/categories",
                """
                {
                  "description": "Saúde",
                  "icon": "heart"
                }
                """);

    assertThat(result.status()).isEqualTo(201);
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

    createCategory(ownerSession, "Categoria do Dono", false);
    createCategory(otherSession, "Categoria de Outro", false);

    var result = authenticated(ownerSession).get("/categories");

    assertThat(result.status()).isEqualTo(200);
    assertThat(result.body()).contains("Categoria do Dono");
    assertThat(result.body()).doesNotContain("Categoria de Outro");
  }

  @Test
  void findCategoryReturnsOwnedCategory() throws Exception {
    var email = "category-find@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);

    var categoryId = createCategory(session, "Lazer", false);

    var result = authenticated(session).get("/categories/" + categoryId);

    assertThat(result.status()).isEqualTo(200);
    assertThat(result.body()).contains("\"id\":\"" + categoryId + "\"");
    assertThat(result.body()).contains("\"description\":\"Lazer\"");
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

    var categoryId = createCategory(ownerSession, "Categoria Privada", false);

    var result = authenticated(otherSession).get("/categories/" + categoryId);

    assertThat(result.status()).isEqualTo(404);
  }

  @Test
  void findUnknownCategoryReturnsNotFound() throws Exception {
    var email = "category-find-unknown@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);

    var result = authenticated(session).get("/categories/" + UUID.randomUUID());

    assertThat(result.status()).isEqualTo(404);
  }

  @Test
  void updateCategoryUpdatesDescriptionAndIcon() throws Exception {
    var email = "category-update@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);

    var categoryId = createCategory(session, "Nome Antigo", false);

    var result =
        authenticated(session)
            .put(
                "/categories/" + categoryId,
                """
                {
                  "description": "Nome Novo",
                  "icon": "new"
                }
                """);

    assertThat(result.status()).isEqualTo(200);
    assertThat(result.body()).contains("\"description\":\"Nome Novo\"");
    assertThat(result.body()).contains("\"icon\":\"new\"");
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

    var categoryId = createCategory(ownerSession, "Categoria do Dono", false);

    var result =
        authenticated(otherSession)
            .put(
                "/categories/" + categoryId,
                """
                {
                  "description": "Tentativa"
                }
                """);

    assertThat(result.status()).isEqualTo(404);
  }

  @Test
  void updateCategoryToDuplicatedDescriptionReturnsBadRequest() throws Exception {
    var email = "category-update-dup@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);

    createCategory(session, "Primeira", false);
    var categoryId = createCategory(session, "Segunda", false);

    var result =
        authenticated(session)
            .put(
                "/categories/" + categoryId,
                """
                {
                  "description": "Primeira"
                }
                """);

    assertThat(result.status()).isEqualTo(400);
    assertThat(result.body()).contains("Você já possui uma categoria com essa descrição");
  }

  @Test
  void updateCategoryWithInvalidDescriptionReturnsBadRequest() throws Exception {
    var email = "category-update-invalid@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);

    var categoryId = createCategory(session, "Válida", false);

    var result =
        authenticated(session)
            .put(
                "/categories/" + categoryId,
                """
                {
                  "description": "Ab"
                }
                """);

    assertThat(result.status()).isEqualTo(400);
    assertThat(result.body()).contains("Descrição da categoria deve ter entre 3 e 100 caracteres");
  }

  @Test
  void deleteCategoryWithoutTransactionsReturnsNoContent() throws Exception {
    var email = "category-delete@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);

    var categoryId = createCategory(session, "Categoria para Deletar", false);

    var result = authenticated(session).delete("/categories/" + categoryId);

    assertThat(result.status()).isEqualTo(204);

    var findResult = authenticated(session).get("/categories/" + categoryId);
    assertThat(findResult.status()).isEqualTo(404);
  }

  @Test
  void deleteCategoryWithTransactionsReturnsConflict() throws Exception {
    var email = "category-delete-with-tx@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);

    var categoryId = createCategory(session, "Categoria com Transação", false);
    var userId = findUserIdByEmail(email);
    var accountId =
        jdbcTemplate.queryForObject(
            "INSERT INTO tb_bank_accounts (id, user_id, name, initial_balance, type) VALUES (uuidv7(), ?, 'Conta Teste', 0, 'CHECKING') RETURNING id",
            UUID.class,
            userId);

    jdbcTemplate.update(
        """
                INSERT INTO tb_transactions (id, user_id, bank_account_id, category_id, description, value, type, status, due_date)
                VALUES (uuidv7(), ?, ?, ?, 'Teste', 50.00, 'FLEXIBLE_EXPENSE', 'CONFIRMED', CURRENT_DATE)
                """,
        userId,
        accountId,
        categoryId);

    var result = authenticated(session).delete("/categories/" + categoryId);

    assertThat(result.status()).isEqualTo(409);
    assertThat(result.body()).contains("Categoria possui transações vinculadas");
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

    var categoryId = createCategory(ownerSession, "Categoria do Dono", false);

    var result = authenticated(otherSession).delete("/categories/" + categoryId);

    assertThat(result.status()).isEqualTo(404);
  }

  @Test
  void unauthenticatedAccessReturnsUnauthorized() {
    var result = restTemplate.getForEntity("/categories", String.class);

    assertThat(result.getStatusCode().value()).isEqualTo(401);
  }
}
