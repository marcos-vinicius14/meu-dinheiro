package com.marcos.meudinheiro.transaction;

import static org.assertj.core.api.Assertions.assertThat;

import com.marcos.meudinheiro.IntegrationTestSupport;
import org.junit.jupiter.api.Test;

class TransactionIT extends IntegrationTestSupport {

  @Test
  void createProjectedTransactionReturnsCreated() throws Exception {
    var email = "tx-create@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);
    var categoryId = createCategory(session, "Alimentação", true);

    var result =
        authenticated(session)
            .post(
                "/transactions",
                """
                {
                  "description": "Supermercado",
                  "amount": 150.50,
                  "type": "FLEXIBLE_EXPENSE",
                  "dueDate": "2026-09-20",
                  "categoryId": "%s"
                }
                """
                    .formatted(categoryId));

    assertThat(result.status()).isEqualTo(201);
    assertThat(result.body()).contains("\"description\":\"Supermercado\"");
    assertThat(result.body()).contains("\"type\":\"FLEXIBLE_EXPENSE\"");
    assertThat(result.body()).contains("\"status\":\"PROJECTED\"");
    assertThat(result.body()).contains("\"dueDate\":\"2026-09-20\"");
  }

  @Test
  void createTransactionWithBankAccountReturnsCreated() throws Exception {
    var email = "tx-create-account@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);
    var categoryId = createCategory(session, "Moradia", false);
    var accountId = createBankAccount(session, "Conta Corrente");

    var result =
        authenticated(session)
            .post(
                "/transactions",
                """
                {
                  "description": "Aluguel",
                  "amount": 1200.00,
                  "type": "FIXED_EXPENSE",
                  "dueDate": "2026-09-10",
                  "categoryId": "%s",
                  "bankAccountId": "%s"
                }
                """
                    .formatted(categoryId, accountId));

    assertThat(result.status()).isEqualTo(201);
    assertThat(result.body()).contains("\"bankAccountId\":\"" + accountId + "\"");
  }

  @Test
  void createTransactionWithoutCategoryReturnsBadRequest() throws Exception {
    var email = "tx-no-category@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);

    var result =
        authenticated(session)
            .post(
                "/transactions",
                """
                {
                  "description": "Sem categoria",
                  "amount": 100.00,
                  "type": "FLEXIBLE_EXPENSE",
                  "dueDate": "2026-09-20"
                }
                """);

    assertThat(result.status()).isEqualTo(400);
  }

  @Test
  void createTransactionWithZeroAmountReturnsBadRequest() throws Exception {
    var email = "tx-zero@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);
    var categoryId = createCategory(session, "Zero", true);

    var result =
        authenticated(session)
            .post(
                "/transactions",
                """
                {
                  "description": "Zero",
                  "amount": 0,
                  "type": "FLEXIBLE_EXPENSE",
                  "dueDate": "2026-09-20",
                  "categoryId": "%s"
                }
                """
                    .formatted(categoryId));

    assertThat(result.status()).isEqualTo(400);
  }

  @Test
  void createTransactionWithInstallmentTypeReturnsBadRequest() throws Exception {
    var email = "tx-installment@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);
    var categoryId = createCategory(session, "Parcela", false);

    var result =
        authenticated(session)
            .post(
                "/transactions",
                """
                {
                  "description": "Parcela avulsa",
                  "amount": 100.00,
                  "type": "INSTALLMENT_EXPENSE",
                  "dueDate": "2026-09-20",
                  "categoryId": "%s"
                }
                """
                    .formatted(categoryId));

    assertThat(result.status()).isEqualTo(400);
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
    var categoryId = createCategory(ownerSession, "Privada", true);

    var result =
        authenticated(otherSession)
            .post(
                "/transactions",
                """
                {
                  "description": "Categoria alheia",
                  "amount": 100.00,
                  "type": "FLEXIBLE_EXPENSE",
                  "dueDate": "2026-09-20",
                  "categoryId": "%s"
                }
                """
                    .formatted(categoryId));

    assertThat(result.status()).isEqualTo(400);
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
    var categoryId = createCategory(ownerSession, "Lista", true);
    var otherCategoryId = createCategory(otherSession, "Lista Alheia", true);

    createTransaction(ownerSession, "Do dono", "100.00", "FLEXIBLE_EXPENSE", categoryId);
    createTransaction(otherSession, "De outro", "200.00", "FLEXIBLE_EXPENSE", otherCategoryId);

    var result = authenticated(ownerSession).get("/transactions");

    assertThat(result.status()).isEqualTo(200);
    assertThat(result.body()).contains("Do dono");
    assertThat(result.body()).doesNotContain("De outro");
  }

  @Test
  void findTransactionReturnsOwned() throws Exception {
    var email = "tx-find@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);
    var categoryId = createCategory(session, "Busca", true);

    var transactionId =
        createTransaction(session, "Para buscar", "75.00", "FLEXIBLE_EXPENSE", categoryId);

    var result = authenticated(session).get("/transactions/" + transactionId);

    assertThat(result.status()).isEqualTo(200);
    assertThat(result.body()).contains("\"id\":\"" + transactionId + "\"");
    assertThat(result.body()).contains("Para buscar");
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
    var categoryId = createCategory(ownerSession, "Privada", true);

    var transactionId =
        createTransaction(ownerSession, "Privada", "10.00", "FLEXIBLE_EXPENSE", categoryId);

    var result = authenticated(otherSession).get("/transactions/" + transactionId);

    assertThat(result.status()).isEqualTo(404);
  }

  @Test
  void updateTransactionChangesFields() throws Exception {
    var email = "tx-update@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);
    var categoryId = createCategory(session, "Update", true);

    var transactionId =
        createTransaction(session, "Antigo", "100.00", "FLEXIBLE_EXPENSE", categoryId);

    var result =
        authenticated(session)
            .put(
                "/transactions/" + transactionId,
                """
                {
                  "description": "Novo",
                  "amount": 250.00,
                  "type": "FIXED_EXPENSE",
                  "dueDate": "2026-09-25",
                  "categoryId": "%s"
                }
                """
                    .formatted(categoryId));

    assertThat(result.status()).isEqualTo(200);
    assertThat(result.body()).contains("\"description\":\"Novo\"");
    assertThat(result.body()).contains("\"amount\":250.00");
    assertThat(result.body()).contains("\"type\":\"FIXED_EXPENSE\"");
    assertThat(result.body()).contains("\"dueDate\":\"2026-09-25\"");
  }

  @Test
  void deleteTransactionReturnsNoContent() throws Exception {
    var email = "tx-delete@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);
    var categoryId = createCategory(session, "Delete", true);

    var transactionId =
        createTransaction(session, "Para deletar", "10.00", "FLEXIBLE_EXPENSE", categoryId);

    var result = authenticated(session).delete("/transactions/" + transactionId);

    assertThat(result.status()).isEqualTo(204);

    var findResult = authenticated(session).get("/transactions/" + transactionId);
    assertThat(findResult.status()).isEqualTo(404);
  }

  @Test
  void createBundleGeneratesCommittedInstallments() throws Exception {
    var email = "tx-bundle@example.com";
    var password = "password123";
    createUser(email, password);
    var session = login(email, password);
    var categoryId = createCategory(session, "Parcelado", false);

    var result =
        authenticated(session)
            .post(
                "/transactions/bundles",
                """
                {
                  "description": "Notebook",
                  "totalAmount": 1000.00,
                  "totalInstallments": 10,
                  "firstDueDate": "2026-09-15",
                  "categoryId": "%s"
                }
                """
                    .formatted(categoryId));

    assertThat(result.status()).isEqualTo(201);
    assertThat(result.body()).contains("\"totalInstallments\":10");

    var bundleTransactions =
        jdbcTemplate.queryForObject(
            "SELECT COUNT(*) FROM tb_transactions WHERE bundle_id IS NOT NULL AND user_id = ?",
            Integer.class,
            findUserIdByEmail(email));

    assertThat(bundleTransactions).isEqualTo(10);
  }

  @Test
  void unauthenticatedAccessReturnsUnauthorized() {
    var result = restTemplate.getForEntity("/transactions", String.class);

    assertThat(result.getStatusCode().value()).isEqualTo(401);
  }
}
