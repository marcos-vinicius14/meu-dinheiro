package com.marcos.meudinheiro.identity.infraestructure.security;

import jakarta.servlet.http.Cookie;
import jakarta.servlet.http.HttpServletRequest;

import java.util.Arrays;

import org.springframework.security.oauth2.server.resource.web.BearerTokenResolver;
import org.springframework.stereotype.Component;

@Component
public class CookieBearerTokenResolver  implements BearerTokenResolver {


    public static final String ACCESS_TOKEN_COOKIE = "access_token";

    @Override
    public String resolve(HttpServletRequest request) {
        var cookies = request.getCookies();

        if (cookies == null) {
            return null;
        }

        return Arrays.stream(cookies)
            .filter(this::isAccessToken)
            .map(cookie -> cookie.getValue())
            .findFirst()
            .orElse(null);
    }

    private boolean isAccessToken(Cookie cookie) {
        return ACCESS_TOKEN_COOKIE.equals(cookie.getName());
    }
    
}
