package com.marcos.meudinheiro;

import java.time.Duration;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.testcontainers.service.connection.ServiceConnection;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.testcontainers.containers.PostgreSQLContainer;

/**
 * Base dos ITs: contexto Spring UNICO compartilhado por todas as suítes (mesmas propriedades =
 * mesmo contexto cacheado) com Postgres real via Testcontainers e servidor HTTP real em porta
 * aleatória.
 *
 * <p>Configurações de segurança: rate limit por IP generoso (1000) e lockout de conta com
 * max-attempts=2. Testes que precisam esgotar limites usam IPs (X-Forwarded-For) ou emails próprios
 * para não interferir nos demais.
 */
@SpringBootTest(
    webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT,
    properties = {
      "spring.docker.compose.enabled=false",
      "spring.jpa.show-sql=false",
      "DATABASE_URL=jdbc:postgresql://localhost:5432/dummy",
      "POSTGRES_USER=dummy",
      "POSTGRES_PASSWORD=dummy",
      "security.login.rate-limit.max-requests=1000",
      "security.login.max-attempts=2"
    })
public abstract class AbstractIntegrationTest {

  @ServiceConnection static final PostgreSQLContainer<?> POSTGRES = startPostgres();

  protected AbstractIntegrationTest() {}

  @DynamicPropertySource
  static void registerProperties(DynamicPropertyRegistry registry) {
    // Ponto de extensão: propriedades dinâmicas por teste futuro (ex.: Clock)
  }

  private static PostgreSQLContainer<?> startPostgres() {
    var container =
        new PostgreSQLContainer<>("postgres:18-alpine").withStartupTimeout(Duration.ofMinutes(3));
    container.start();
    return container;
  }
}
