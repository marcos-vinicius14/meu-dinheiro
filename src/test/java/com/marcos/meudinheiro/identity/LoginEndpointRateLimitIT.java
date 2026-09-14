package com.marcos.meudinheiro.identity;

import static org.assertj.core.api.Assertions.assertThat;

import com.marcos.meudinheiro.IntegrationTestSupport;
import org.junit.jupiter.api.Test;
import org.springframework.http.HttpEntity;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpMethod;
import org.springframework.test.context.TestPropertySource;

/**
 * Rate limit por IP no endpoint. Cada teste usa um IP proprio via header
 * X-Forwarded-For (cenario real: cliente atras de proxy), isolando os contadores
 * entre testes sem contexto Spring separado.
 */
@TestPropertySource(properties = "security.login.rate-limit.max-requests=3")
class LoginEndpointRateLimitIT extends IntegrationTestSupport {

    @Test
    void fourthLoginAttemptFromSameIpReturnsTooManyRequests() throws Exception {
        var password = "password123";
        var ip = "10.1.0.1";

        // Emails distintos: lockout (por conta) não contamina o teste de IP.
        loginFrom(ip, "ip-ratelimit-blocked-1@example.com", "wrong-password", 401);
        loginFrom(ip, "ip-ratelimit-blocked-2@example.com", "wrong-password", 401);
        loginFrom(ip, "ip-ratelimit-blocked-3@example.com", "wrong-password", 401);

        var email = "ip-ratelimit-blocked-4@example.com";
        createUser(email, password);
        var result = loginFrom(ip, email, password, 429);
        assertThat(result).contains("errors");
    }

    @Test
    void validLoginUnderLimitSucceeds() throws Exception {
        var email = "ip-ratelimit-under-limit@example.com";
        var password = "password123";

        createUser(email, password);

        loginFrom("10.1.0.2", email, password, 204);
    }

    @Test
    void limitIsPerIpNotGlobal() throws Exception {
        var password = "password123";
        var exhaustedIp = "10.1.0.3";
        var otherIp = "10.1.0.4";

        // Emails distintos: o lockout (max-attempts) é por CONTA e não pode
        // contaminar este teste, que valida apenas o limite por IP.
        loginFrom(exhaustedIp, "ip-ratelimit-per-ip-1@example.com", "wrong-password", 401);
        loginFrom(exhaustedIp, "ip-ratelimit-per-ip-2@example.com", "wrong-password", 401);
        loginFrom(exhaustedIp, "ip-ratelimit-per-ip-3@example.com", "wrong-password", 401);

        var email = "ip-ratelimit-per-ip-free@example.com";
        createUser(email, password);
        loginFrom(otherIp, email, password, 204);
    }

    private String loginFrom(String ip, String email, String password, int expectedStatus) {
        var headers = new HttpHeaders();
        headers.add("X-Forwarded-For", ip);
        headers.add(HttpHeaders.CONTENT_TYPE, "application/json");

        var entity = new HttpEntity<>(loginBody(email, password), headers);
        var response = restTemplate.exchange(
                "/auth/login",
                HttpMethod.POST,
                entity,
                String.class
        );

        assertThat(response.getStatusCode().value())
                .as("IP %s login %s", ip, email)
                .isEqualTo(expectedStatus);
        return response.getBody();
    }

    private String loginBody(String email, String password) {
        return """
                {
                  "email": "%s",
                  "password": "%s"
                }
                """.formatted(email, password);
    }
}
