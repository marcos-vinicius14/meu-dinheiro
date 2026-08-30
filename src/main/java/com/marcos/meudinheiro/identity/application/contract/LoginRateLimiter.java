package com.marcos.meudinheiro.identity.application.contract;

public interface LoginRateLimiter {
    boolean tryAcquire(String clientIp);
}
