package com.marcos.meudinheiro.identity;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import org.junit.jupiter.api.Test;
import org.springframework.http.MediaType;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.web.servlet.request.RequestPostProcessor;

/**
 * Rate limit por IP no endpoint. O limitador e um bean singleton compartilhado
 * entre os metodos de teste (mesmo contexto): cada teste usa um IP fonte
 * proprio para nao interferir nos demais.
 */
@TestPropertySource(properties = "security.login.rate-limit.max-requests=3")
class LoginEndpointRateLimitIT extends AuthenticationTestSupport {

    @Test
    void fourthLoginAttemptFromSameIpReturnsTooManyRequests() throws Exception {
        var email = "ip-ratelimit-blocked@example.com";
        var password = "password123";
        var ip = clientIp("10.1.0.1");

        createUser(email, password);

        loginFrom(ip, email, "wrong-password", 401);
        loginFrom(ip, email, "wrong-password", 401);
        loginFrom(ip, email, "wrong-password", 401);

        mockMvc.perform(post("/auth/login")
                        .with(ip)
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(loginBody(email, password)))
                .andExpect(status().isTooManyRequests())
                .andExpect(jsonPath("$.errors").isArray());
    }

    @Test
    void validLoginUnderLimitSucceeds() throws Exception {
        var email = "ip-ratelimit-under-limit@example.com";
        var password = "password123";
        var ip = clientIp("10.1.0.2");

        createUser(email, password);

        loginFrom(ip, email, password, 204);
    }

    @Test
    void limitIsPerIpNotGlobal() throws Exception {
        var email = "ip-ratelimit-per-ip@example.com";
        var password = "password123";
        var exhaustedIp = clientIp("10.1.0.3");
        var otherIp = clientIp("10.1.0.4");

        createUser(email, password);

        loginFrom(exhaustedIp, email, "wrong-password", 401);
        loginFrom(exhaustedIp, email, "wrong-password", 401);
        loginFrom(exhaustedIp, email, "wrong-password", 401);

        loginFrom(otherIp, email, password, 204);
    }

    private RequestPostProcessor clientIp(String ip) {
        return request -> {
            request.setRemoteAddr(ip);
            return request;
        };
    }

    private void loginFrom(
            RequestPostProcessor clientIp,
            String email,
            String password,
            int expectedStatus
    ) throws Exception {
        mockMvc.perform(post("/auth/login")
                        .with(clientIp)
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(loginBody(email, password)))
                .andExpect(status().is(expectedStatus));
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
