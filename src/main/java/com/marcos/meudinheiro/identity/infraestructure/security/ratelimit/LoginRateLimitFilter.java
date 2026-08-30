package com.marcos.meudinheiro.identity.infraestructure.security.ratelimit;

import java.io.IOException;
import java.util.List;

import org.springframework.http.HttpMethod;
import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.stereotype.Component;
import org.springframework.web.filter.OncePerRequestFilter;

import tools.jackson.databind.ObjectMapper;

import com.marcos.meudinheiro.identity.application.contract.LoginRateLimiter;
import com.marcos.meudinheiro.shared.infraestructure.web.response.ErrorResponse;

import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;

@Component
public class LoginRateLimitFilter extends OncePerRequestFilter {

    private final LoginRateLimiter rateLimiter;
    private final ObjectMapper objectMapper;

    public LoginRateLimitFilter(
        LoginRateLimiter rateLimiter,
        ObjectMapper objectMapper
    ) {
        this.rateLimiter = rateLimiter;
        this.objectMapper = objectMapper;
    }

    @Override
    protected boolean shouldNotFilter(HttpServletRequest request) {
        return !HttpMethod.POST.matches(request.getMethod())
                || !"/auth/login".equals(request.getRequestURI());
    }

    @Override
    protected void doFilterInternal(
        HttpServletRequest request,
        HttpServletResponse response,
        FilterChain filterChain
    ) throws ServletException, IOException {

        if (!rateLimiter.tryAcquire(request.getRemoteAddr())) {
            response.setStatus(HttpStatus.TOO_MANY_REQUESTS.value());
            response.setContentType(MediaType.APPLICATION_JSON_VALUE);

            objectMapper.writeValue(
                response.getWriter(),
                new ErrorResponse(List.of("Muitas requisições. Tente novamente mais tarde"))
            );

            return;
        }

        filterChain.doFilter(request, response);
    }
}
