package com.marcos.meudinheiro.identity.application.contract;

public interface LoginRateLimiter {
    boolean tryAcquire(String clientIp);

    /**
     * Limpa todos os contadores (uso operacional/testes; janela expira
     * naturalmente em produção).
     */
    void reset();
}
