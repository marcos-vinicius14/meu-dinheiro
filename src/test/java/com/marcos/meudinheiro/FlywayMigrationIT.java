package com.marcos.meudinheiro;

import static org.assertj.core.api.Assertions.assertThat;

import javax.sql.DataSource;

import org.flywaydb.core.Flyway;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;

/**
 * Valida que a sequência completa de migrations (V1..Vn) é re-executável do
 * zero: derruba o schema, roda o migrate limpo e confere o estado final.
 * Complementa o contexto principal (onde o Flyway roda uma única vez sobre
 * um banco já migrado ou vazio no boot do container).
 */
class FlywayMigrationIT extends AbstractIntegrationTest {

    @Autowired
    private DataSource dataSource;

    @Test
    void migrationsAreReexecutableFromScratch() {
        var flyway = Flyway.configure()
                .dataSource(dataSource)
                .locations("classpath:db/migration")
                .cleanDisabled(false)
                .load();

        flyway.clean();
        var result = flyway.migrate();

        assertThat(result.success).isTrue();
        assertThat(result.migrationsExecuted).isGreaterThan(0);

        var applied = new org.springframework.jdbc.core.JdbcTemplate(dataSource)
                .queryForObject(
                        "SELECT COUNT(*) FROM flyway_schema_history WHERE success = true",
                        Integer.class
                );

        assertThat(applied).isEqualTo(result.migrationsExecuted);
    }
}
