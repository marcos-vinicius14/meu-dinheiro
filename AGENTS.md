# AGENTS.md

## Comandos essenciais

Requer **JDK 25** — o default do shell é 21 e falha com `release version 25 not supported`:

```bash
JAVA_HOME=~/.sdkman/candidates/java/25.0.1-graalce ./mvnw verify   # tudo: compile + unit + IT
JAVA_HOME=~/.sdkman/candidates/java/25.0.1-graalce ./mvnw test -Dtest=RefreshTokenModelTest          # um unitário
JAVA_HOME=~/.sdkman/candidates/java/25.0.1-graalce ./mvnw test-compile failsafe:integration-test -Dit.test=LoginIT  # um IT
```

- Docker obrigatório para ITs (Testcontainers). Disco cheio quebra o startup do container com timeout genérico — cheque `df -h` antes de debugar.
- Convenção de suítes: `*Test` → surefire (sem Spring); `*IT` → failsafe (sobe contexto + Postgres).
- Não há script lint/typecheck; verificação = `mvnw verify` compilar e passar.

## Arquitetura

Modular monolith hexagonal (veja `docs/roadmap.md` — v1). Módulos sob `src/main/java/com/marcos/meudinheiro/`: `identity`, `user`, `category`, `transaction`, `bankaccount`, `shared`.

- Layout por módulo: `application/contract` (ports + DTOs), `application/usecase`, `domain/{model,valueobject,enums}`, `infraestructure/{web,repository,security,configuration}`.
- **Atenção ao typo**: o módulo `user` usa o pacote `infraesctructure` (com "c" extra); os demais usam `infraestructure`. Não "corrija" — imports existentes dependem disso.
- Use cases consomem ports de outros módulos (ex.: identity usa `FindUserIdentityUseCase` do user) — a comunicação inter-módulos é via application/contract, nunca via repository alheio.

## Notification pattern (obrigatório)

Exceção só para caso excepcional (infra caída, contrato de framework, bug). Violação de regra de negócio retorna `OperationResult<T>` via `shared/notification` — detalhes e exemplos em `docs/notification-pattern.md`.

- `ValidationResult<T>` (uma validação) → `Notification.collect()` (acumula) → `OperationResult<T>` (retorno do use case).
- Controller decide status HTTP; use case não conhece HTTP.
- Mensagens de erro em português; em segurança, mensagens opacas (não revele se token existe/expirou/foi revogado).

## Optional / Maybe Type para ausência de valor

Evite atribuir `null` a variáveis para representar "não encontrado". Prefira encadear `Optional` e tratar a ausência de valor no final do pipeline.

- Use `findById(...)` (ou métodos de repository que já retornam `Optional`) e encadene `.filter(...)`, `.map(...)` e `.orElseGet(...)`.
- Não use `.orElse(null)` seguido de `if (obj == null)` — isso recria o mesmo problema que o `Optional` resolve.
- Para lógicas de sucesso maiores (ex.: validação de input antes de salvar), extraia um método privado e chame dentro do `.map(...)`.

Exemplo:

```java
return repository.findById(accountId)
        .filter(account -> account.belongsTo(userId))
        .map(BankAccountMapper::toOutput)
        .map(OperationResult::success)
        .orElseGet(() -> OperationResult.failure("Conta não encontrada"));
```

```java
return repository.findById(accountId)
        .filter(account -> account.belongsTo(userId))
        .map(account -> updateAccount(account, input))
        .orElseGet(() -> OperationResult.failure("Conta não encontrada"));
```

## Armadilhas verificadas (custaram debugging)

- **`final class` + `@Transactional` não funciona**: CGLIB não cria proxy → `AopConfigException` no boot. Use cases transacionais são `public class` não-final.
- **Testcontainers**: `AbstractIntegrationTest` usa **singleton container** (start manual no static initializer, SEM `@Testcontainers`/`@Container`). Re-adicionar as anotações para o container entre classes e mata o contexto cacheado (classes seguintes falham com 500/timeout).
- **`.class` obsoleto em `target/`**: delete de fonte após edição sem clean pode deixar classe velha em `target/classes` → `ConflictingBeanDefinitionException` no boot. Se der conflito de bean duplicado, `rm -rf target/classes` antes de investigar o código.
- **Cache Caffeine compartilhado em ITs**: o lockout do login (`CaffeineLoginAttemptLimiter`) é um bean singleton compartilhado por contexto — testes de lockout precisam de emails únicos E `@TestPropertySource` (contexto novo) para `max-attempts` baixo; não use o mesmo email em testes de lockout e de login normal. O mesmo vale para o rate limit por IP (`LoginRateLimitFilter`): `AbstractIntegrationTest` desabilita via `max-requests=1000`, e cada teste de `LoginEndpointRateLimitIT` usa IP fonte próprio (via `request.setRemoteAddr`) para não interferir entre métodos.
- **Jackson 3 (Boot 4)**: ObjectMapper é `tools.jackson.databind.ObjectMapper` — `com.fasterxml.jackson.databind` não existe no classpath.
- **Chaves RSA**: app não boota sem `classpath:keys/*.pem`. As de teste estão em `src/test/resources/keys/`; as de runtime (`src/main/resources/keys/`) **não estão commitadas** — gere localmente para rodar a app fora de teste.
- **Spring Boot 4.x**: nomes de artefatos mudaram (`spring-boot-starter-webmvc`, `org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc`) e o parent NÃO gerencia `org.testcontainers:junit-jupiter`/`:postgresql` (Testcontainers 2.x usa `testcontainers-junit-jupiter`, `testcontainers-postgresql`).
- **Banco**: `ddl-auto: validate` — Flyway (`src/main/resources/db/migration`) é a fonte de verdade; entidade nova sem migration falha no boot.
- **`revoke`/`delete` não são verbos deriváveis do Spring Data**: bulk update/delete em repository precisa `@Modifying @Query` (JPQL) — ver `RefreshTokenRepository`.

## Testes (estratégia: Testing Trophy)

Referência completa em `docs/testing-strategy.md`. Resumo operacional:

- Corpo principal são ITs: `@SpringBootTest` + `@AutoConfigureMockMvc` + Postgres real. Mock só para dependências não-determinísticas; preferir manipular estado real (ex.: UPDATE no `expires_at` para simular expiração).
- Estenda `AbstractIntegrationTest` (módulo identity: `AuthenticationTestSupport`, com helpers `createUser`/`login`/`Session`/`jdbcTemplate`).
- ITs compartilham banco e contexto: cada teste cria dados próprios com **emails únicos**; sem `@Transactional` de teste (commits reais validam SQL de verdade).

## TDD (obrigatório para nova funcionalidade)

Ciclo vermelho → verde por feature, na ordem:

1. **RED**: escreva o IT primeiro, expressando o comportamento CORRETO (não o que o código hoje faz). Rode e veja falhar — a falha confirma que o teste testifica algo novo (`./mvnw test-compile failsafe:integration-test -Dit.test=XxxIT`).
2. **GREEN**: implemente o mínimo para passar. Use case segue notification pattern (`OperationResult`); controller só mapeia HTTP.
3. Comportamento mudou de propósito (novo contrato)? Atualize o teste antigo para o novo contrato — não contorne a feature nova no teste.

Casos reais onde o vermelho pegou bug antes do código: login devolvia 500 (não 401) em credenciais inválidas; cleanup ignorava retenção (`expiresAt < now` em vez de `now - retention`); reuso de refresh não revogava a cadeia.

Armadilha TDD: asserção fraca passa sem testar nada — assert só no que é garantido por design (ex.: JWTs gerados no mesmo segundo com mesmo subject são idênticos; a rotação garantida é a do refresh opaco).

## Runtime local

- `compose.yaml` sobe Postgres 18; credenciais via `.env` (`DATABASE_URL`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` — não commitado).
- Cookies de auth: `access_token` path `/` (JWT, 10 min), `refresh_token` path `/auth` (opaco, hash SHA-256 no banco, TTL 15 dias, rotação a cada uso). Ao expirar cookie no logout, repita path/atributos exatos ou o browser não deleta.
- Segurança (lockout, detecção de reuso, cleanup, decisões): ver `docs/security.md`. Reuso de refresh revogado derruba TODAS as sessões do usuário (roubo presumido).

## Git / PR

- Commits em português ou inglês, conventional commits (`feat(identity): ...`, `refactor: ...`).
- Nunca commite: `.env`, `src/main/resources/keys/`, chaves de qualquer tipo.
