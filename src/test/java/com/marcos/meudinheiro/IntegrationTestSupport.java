package com.marcos.meudinheiro;

import static org.assertj.core.api.Assertions.assertThat;

import java.util.UUID;

import org.junit.jupiter.api.AfterAll;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.resttestclient.autoconfigure.AutoConfigureTestRestTemplate;
import org.springframework.boot.resttestclient.TestRestTemplate;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.jdbc.core.JdbcTemplate;
import jakarta.servlet.http.Cookie;
import tools.jackson.databind.JsonNode;
import tools.jackson.databind.ObjectMapper;

/**
 * Suporte compartilhado para ITs: autenticação real, helpers de criação de
 * recursos via API e extração de campos JSON sem parsing por string.
 *
 * HTTP: {@link #restTemplate} fala com o servidor real (RANDOM_PORT, Tomcat).
 */
@AutoConfigureTestRestTemplate
public abstract class IntegrationTestSupport extends AbstractIntegrationTest {

    protected static final String ACCESS_TOKEN_COOKIE = "access_token";
    protected static final String REFRESH_TOKEN_COOKIE = "refresh_token";

    @Autowired
    protected JdbcTemplate jdbcTemplate;

    @Autowired
    protected ObjectMapper objectMapper;

    @Autowired
    protected TestRestTemplate restTemplate;

    /**
     * Limpeza de dados de domínio após cada classe de teste: mantém o container
     * singleton previsível entre suítes sem recriar o contexto Spring.
     * Preserva tb_users? Não — remove usuários e tudo que casca em cascata.
     * Chaves/segurança não são tocadas.
     */
    @AfterAll
    static void cleanDatabase(@Autowired JdbcTemplate jdbcTemplate) {
        jdbcTemplate.execute("""
                TRUNCATE TABLE tb_transactions, tb_transaction_bundles, tb_check_in_snapshots,
                               tb_bank_accounts, tb_categories, tb_refresh_tokens, tb_users
                RESTART IDENTITY CASCADE
                """);
    }

    public record Session(String accessToken, String refreshToken) {
        public Cookie accessTokenCookie() {
            return new Cookie(ACCESS_TOKEN_COOKIE, accessToken);
        }

        public Cookie refreshTokenCookie() {
            return new Cookie(REFRESH_TOKEN_COOKIE, refreshToken);
        }
    }

    // ---- Autenticação ----

    protected void createUser(String email, String password) {
        var response = restTemplate.postForEntity(
                "/users",
                new org.springframework.http.HttpEntity<>(
                        """
                                {
                                  "username": "Test User",
                                  "email": "%s",
                                  "password": "%s"
                                }
                                """.formatted(email, password),
                        jsonHeaders()
                ),
                String.class
        );

        assertThat(response.getStatusCode().value()).isEqualTo(204);
    }

    protected Session login(String email, String password) {
        return loginReal(email, password);
    }

    /**
     * Login via HTTP real (porta aleatória): captura os Set-Cookie emitidos
     * pelo Tomcat. Use este helper quando o teste valida o comportamento do
     * endpoint em si (cookies, headers, rotação).
     */
    protected Session loginReal(String email, String password) {
        var response = restTemplate.postForEntity(
                "/auth/login",
                new org.springframework.http.HttpEntity<>(
                        """
                                {
                                  "email": "%s",
                                  "password": "%s"
                                }
                                """.formatted(email, password),
                        jsonHeaders()
                ),
                String.class
        );

        assertThat(response.getStatusCode().value()).isEqualTo(204);

        var cookies = response.getHeaders().get(HttpHeaders.SET_COOKIE);
        return new Session(
                cookieFromHeader(cookies, ACCESS_TOKEN_COOKIE),
                cookieFromHeader(cookies, REFRESH_TOKEN_COOKIE)
        );
    }

    protected org.springframework.http.HttpHeaders jsonHeaders() {
        var headers = new org.springframework.http.HttpHeaders();
        headers.add(HttpHeaders.CONTENT_TYPE, MediaType.APPLICATION_JSON_VALUE);
        return headers;
    }

    private static String cookieFromHeader(
            java.util.List<String> setCookies,
            String name
    ) {
        return setCookies.stream()
                .filter(header -> header.startsWith(name + "="))
                .map(header -> header.substring((name + "=").length()).split(";", 2)[0])
                .findFirst()
                .orElseThrow(() -> new AssertionError(name + " cookie ausente"));
    }


    private static java.util.stream.Stream<String> extractCookieValue(
            String name,
            String setCookieHeader
    ) {
        if (!setCookieHeader.startsWith(name + "=")) {
            return java.util.stream.Stream.empty();
        }

        var value = setCookieHeader
                .substring((name + "=").length());

        return java.util.stream.Stream.of(value.split(";", 2)[0]);
    }

    // ---- Recursos de domínio (via API real) ----

    protected UUID createCategory(
            Session session,
            String description,
            boolean flexible
    ) {
        var response = authenticated(session).post("/categories", """
                {
                  "description": "%s",
                  "icon": "tag",
                  "isFlexible": %s
                }
                """.formatted(description, flexible));
        assertThat(response.status()).isEqualTo(201);
        return response.id();
    }

    protected UUID createBankAccount(Session session, String name) {
        var response = authenticated(session).post("/bankaccounts", """
                {
                  "name": "%s",
                  "type": "CHECKING"
                }
                """.formatted(name));
        assertThat(response.status()).isEqualTo(201);
        return response.id();
    }

    protected UUID createAccount(
            Session session,
            String name,
            String type,
            String initialBalance
    ) {
        var response = authenticated(session).post("/bankaccounts", """
                {
                  "name": "%s",
                  "type": "%s",
                  "initialBalance": %s
                }
                """.formatted(name, type, initialBalance));
        assertThat(response.status()).isEqualTo(201);
        return response.id();
    }

    protected UUID createCategoryViaSql(UUID userId) {
        return jdbcTemplate.queryForObject(
                "INSERT INTO tb_categories (id, user_id, description, icon, is_flexible) VALUES (uuidv7(), ?, 'Categoria Teste', 'icon', false) RETURNING id",
                UUID.class,
                userId
        );
    }

    protected UUID createTransaction(
            Session session,
            String description,
            String amount,
            String type,
            UUID categoryId
    ) {
        return createTransactionWithDueDate(
                session, description, amount, type, "2026-09-20", categoryId);
    }

    protected UUID createTransactionWithDueDate(
            Session session,
            String description,
            String amount,
            String type,
            String dueDate,
            UUID categoryId
    ) {
        var response = authenticated(session).post("/transactions", """
                {
                  "description": "%s",
                  "amount": %s,
                  "type": "%s",
                  "dueDate": "%s",
                  "categoryId": "%s"
                }
                """.formatted(description, amount, type, dueDate, categoryId));
        assertThat(response.status()).isEqualTo(201);
        return response.id();
    }

    protected UUID findUserIdByEmail(String email) {
        return jdbcTemplate.queryForObject(
                "SELECT id FROM tb_users WHERE email = ?",
                UUID.class,
                email
        );
    }

    // ---- HTTP real (RANDOM_PORT) ----

    /**
     * Request autenticada contra o servidor real com os cookies da sessão.
     */
    protected AuthenticatedRequest authenticated(Session session) {
        return new AuthenticatedRequest(
                ACCESS_TOKEN_COOKIE + "=" + session.accessToken());
    }

    /**
     * Request com cookies crus (ex.: token forjado/expirado em testes de segurança).
     */
    protected AuthenticatedRequest withCookie(String name, String value) {
        return new AuthenticatedRequest(name + "=" + value);
    }

    public class AuthenticatedRequest {
        private final org.springframework.http.HttpHeaders headers = new org.springframework.http.HttpHeaders();

        AuthenticatedRequest(String cookieHeader) {
            headers.add(HttpHeaders.COOKIE, cookieHeader);
            headers.add(HttpHeaders.CONTENT_TYPE, MediaType.APPLICATION_JSON_VALUE);
        }

        public HttpResult post(String uri, String body) {
            return exchange(org.springframework.http.HttpMethod.POST, uri, body);
        }

        public HttpResult get(String uri) {
            return exchange(org.springframework.http.HttpMethod.GET, uri, null);
        }

        public HttpResult put(String uri, String body) {
            return exchange(org.springframework.http.HttpMethod.PUT, uri, body);
        }

        public HttpResult delete(String uri) {
            return exchange(org.springframework.http.HttpMethod.DELETE, uri, null);
        }

        private HttpResult exchange(
                org.springframework.http.HttpMethod method,
                String uri,
                String body
        ) {
            var entity = new org.springframework.http.HttpEntity<>(body, headers);
            var response = restTemplate.exchange(uri, method, entity, String.class);
            return new HttpResult(
                    response.getStatusCode().value(),
                    response.getBody(),
                    response.getHeaders()
            );
        }
    }

    public class HttpResult {
        private final int status;
        private final String body;
        private final org.springframework.http.HttpHeaders responseHeaders;

        HttpResult(int status, String body, org.springframework.http.HttpHeaders responseHeaders) {
            this.status = status;
            this.body = body;
            this.responseHeaders = responseHeaders;
        }

        public int status() {
            return status;
        }

        public String body() {
            return body;
        }

        public java.util.List<String> setCookies() {
            return responseHeaders.get(HttpHeaders.SET_COOKIE);
        }

        public UUID id() {
            return extractId(body);
        }
    }

    // ---- JSON ----

    protected UUID extractId(String responseBody) {
        return extractUuidField(responseBody, "id");
    }

    protected UUID extractUuidField(String responseBody, String field) {
        JsonNode node = objectMapper.readTree(responseBody);
        var value = node.path(field).asText(null);
        if (value == null) {
            throw new AssertionError("Campo ausente na resposta: " + field);
        }
        return UUID.fromString(value);
    }
}
