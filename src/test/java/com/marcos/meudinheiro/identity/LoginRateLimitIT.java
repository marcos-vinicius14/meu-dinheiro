package com.marcos.meudinheiro.identity;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import org.junit.jupiter.api.Test;
import org.springframework.test.context.TestPropertySource;
import org.springframework.http.MediaType;

@TestPropertySource(properties = "security.login.max-attempts=2")
class LoginRateLimitIT extends AuthenticationTestSupport {

    @Test
    void loginBlockedAfterMaxFailedAttemptsEvenWithCorrectPassword() throws Exception {
        var email = "ratelimit-blocked@example.com";
        var password = "password123";

        createUser(email, password);

        failedLogin(email);
        failedLogin(email);

        mockMvc.perform(post("/auth/login")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(loginBody(email, password)))
                .andExpect(status().isUnauthorized())
                .andExpect(jsonPath("$.errors[0]").value("Credenciais inválidas"));
    }

    @Test
    void blockedLoginReturnsSameMessageAsWrongCredentials() throws Exception {
        var email = "ratelimit-opaque@example.com";
        var password = "password123";

        createUser(email, password);

        failedLogin(email);
        failedLogin(email);

        var blockedResponse = mockMvc.perform(post("/auth/login")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(loginBody(email, password)))
                .andExpect(status().isUnauthorized())
                .andReturn().getResponse().getContentAsString();

        var wrongPasswordResponse = mockMvc.perform(post("/auth/login")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(loginBody("ratelimit-opaque-2@example.com", "wrong-pass")))
                .andExpect(status().isUnauthorized())
                .andReturn().getResponse().getContentAsString();

        org.assertj.core.api.Assertions.assertThat(blockedResponse)
                .isEqualTo(wrongPasswordResponse);
    }

    @Test
    void failureCounterResetsAfterSuccessfulLogin() throws Exception {
        var email = "ratelimit-reset@example.com";
        var password = "password123";

        createUser(email, password);

        failedLogin(email);
        login(email, password);
        failedLogin(email);

        mockMvc.perform(post("/auth/login")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(loginBody(email, password)))
                .andExpect(status().isNoContent());
    }

    @Test
    void blockDoesNotAffectOtherAccounts() throws Exception {
        var blockedEmail = "ratelimit-blocked-other@example.com";
        var freeEmail = "ratelimit-free@example.com";
        var password = "password123";

        createUser(blockedEmail, password);
        createUser(freeEmail, password);

        failedLogin(blockedEmail);
        failedLogin(blockedEmail);

        mockMvc.perform(post("/auth/login")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(loginBody(freeEmail, password)))
                .andExpect(status().isNoContent());
    }

    private void failedLogin(String email) throws Exception {
        mockMvc.perform(post("/auth/login")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(loginBody(email, "wrong-password")))
                .andExpect(status().isUnauthorized());
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
