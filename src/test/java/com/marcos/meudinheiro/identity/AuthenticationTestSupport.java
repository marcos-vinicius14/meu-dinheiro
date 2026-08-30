package com.marcos.meudinheiro.identity;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import java.util.stream.Stream;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.MvcResult;

import com.marcos.meudinheiro.AbstractIntegrationTest;

import jakarta.servlet.http.Cookie;

@AutoConfigureMockMvc
public abstract class AuthenticationTestSupport extends AbstractIntegrationTest {

    protected static final String ACCESS_TOKEN_COOKIE = "access_token";
    protected static final String REFRESH_TOKEN_COOKIE = "refresh_token";

    @Autowired
    protected MockMvc mockMvc;

    @Autowired
    protected JdbcTemplate jdbcTemplate;

    protected record Session(String accessToken, String refreshToken) {
        Cookie accessTokenCookie() {
            return new Cookie(ACCESS_TOKEN_COOKIE, accessToken);
        }

        Cookie refreshTokenCookie() {
            return new Cookie(REFRESH_TOKEN_COOKIE, refreshToken);
        }
    }

    protected void createUser(String email, String password) throws Exception {
        mockMvc.perform(post("/users")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "username": "Test User",
                                  "email": "%s",
                                  "password": "%s"
                                }
                                """.formatted(email, password)))
                .andExpect(status().isNoContent());
    }

    protected Session login(String email, String password) throws Exception {
        var result = mockMvc.perform(post("/auth/login")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "email": "%s",
                                  "password": "%s"
                                }
                                """.formatted(email, password)))
                .andExpect(status().isNoContent())
                .andReturn();

        return new Session(
                cookieValue(result, ACCESS_TOKEN_COOKIE),
                cookieValue(result, REFRESH_TOKEN_COOKIE)
        );
    }

    protected String cookieValue(MvcResult result, String name) {
        return result.getResponse()
                .getHeaders(HttpHeaders.SET_COOKIE)
                .stream()
                .flatMap(header -> extractCookieValue(name, header))
                .findFirst()
                .orElseThrow(() ->
                        new AssertionError(name + " cookie ausente")
                );
    }

    private static Stream<String> extractCookieValue(
            String name,
            String setCookieHeader
    ) {
        if (!setCookieHeader.startsWith(name + "=")) {
            return Stream.empty();
        }

        var value = setCookieHeader
                .substring((name + "=").length());

        return Stream.of(value.split(";", 2)[0]);
    }
}
