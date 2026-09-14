package com.marcos.meudinheiro.identity;

import static org.assertj.core.api.Assertions.assertThat;

import org.junit.jupiter.api.Test;
import org.springframework.http.HttpEntity;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpMethod;

import com.marcos.meudinheiro.IntegrationTestSupport;

class LoginRateLimitIT extends IntegrationTestSupport {

    @Test
    void loginBlockedAfterMaxFailedAttemptsEvenWithCorrectPassword() {
        var email = "ratelimit-blocked@example.com";
        var password = "password123";

        createUser(email, password);

        failedLogin(email);
        failedLogin(email);

        var result = login(email, password, 401);

        assertThat(result).contains("Credenciais inválidas");
    }

    @Test
    void blockedLoginReturnsSameMessageAsWrongCredentials() {
        var email = "ratelimit-opaque@example.com";
        var password = "password123";

        createUser(email, password);

        failedLogin(email);
        failedLogin(email);

        var blockedResponse = login(email, password, 401);
        var wrongPasswordResponse = login("ratelimit-opaque-2@example.com", "wrong-pass", 401);

        assertThat(blockedResponse).isEqualTo(wrongPasswordResponse);
    }

    @Test
    void failureCounterResetsAfterSuccessfulLogin() {
        var email = "ratelimit-reset@example.com";
        var password = "password123";

        createUser(email, password);

        failedLogin(email);
        login(email, password);
        failedLogin(email);

        login(email, password, 204);
    }

    @Test
    void blockDoesNotAffectOtherAccounts() {
        var blockedEmail = "ratelimit-blocked-other@example.com";
        var freeEmail = "ratelimit-free@example.com";
        var password = "password123";

        createUser(blockedEmail, password);
        createUser(freeEmail, password);

        failedLogin(blockedEmail);
        failedLogin(blockedEmail);

        login(freeEmail, password, 204);
    }

    private void failedLogin(String email) {
        login(email, "wrong-password", 401);
    }

    private String login(String email, String password, int expectedStatus) {
        var headers = new HttpHeaders();
        headers.add(HttpHeaders.CONTENT_TYPE, "application/json");

        var response = restTemplate.exchange(
                "/auth/login",
                HttpMethod.POST,
                new HttpEntity<>(loginBody(email, password), headers),
                String.class
        );

        assertThat(response.getStatusCode().value())
                .as("login %s", email)
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
