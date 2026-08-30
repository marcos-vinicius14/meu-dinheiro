package com.marcos.meudinheiro;

import java.time.Duration;

import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.testcontainers.service.connection.ServiceConnection;
import org.testcontainers.containers.PostgreSQLContainer;

@SpringBootTest(properties = {
        "spring.docker.compose.enabled=false",
        "spring.jpa.show-sql=false",
        "DATABASE_URL=jdbc:postgresql://localhost:5432/dummy",
        "POSTGRES_USER=dummy",
        "POSTGRES_PASSWORD=dummy",
        "security.login.rate-limit.max-requests=1000"
})
public abstract class AbstractIntegrationTest {

    @ServiceConnection
    static final PostgreSQLContainer<?> POSTGRES = startPostgres();

    protected AbstractIntegrationTest() {
    }

    private static PostgreSQLContainer<?> startPostgres() {
        var container = new PostgreSQLContainer<>("postgres:18-alpine")
                .withStartupTimeout(Duration.ofMinutes(3));
        container.start();
        return container;
    }
}
