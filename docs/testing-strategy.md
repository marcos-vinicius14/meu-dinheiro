# Estratégia de Testes — Testing Trophy

Este projeto adota a **Testing Trophy** (Kent C. Dodds): o corpo principal da pirâmide são os
**testes de integração**, não os unitários com mocks.

```
        /  E2E  \          ← poucos; ITs HTTP com filter chain real já cobrem a maior parte
       /--------\
      /   ITs    \         ← FOCO: MockMvc + Testcontainers (Postgres real)
     /------------\
    /  Unitários   \       ← poucos; apenas lógica de domínio pura (VOs, modelos)
   /----------------\
  /  Tipagem estática \    ← Java + Bean Validation
 /--------------------\
```

## Camadas

| Camada | Ferramenta | Quando usar |
|---|---|---|
| Tipagem estática | Java 25 + jakarta.validation (`@Valid`, `@NotBlank`, `@Email`) | Invariantes de forma de entrada |
| Unitários | JUnit 5 + AssertJ, sem Spring | Lógica de domínio pura e determinística (ex.: `RefreshTokenModelTest`, `RefreshTokenHashTest`) |
| Integração | `@SpringBootTest` + `@AutoConfigureMockMvc` + Testcontainers | Regra geral: use cases, controllers, repositories, filter chain, Flyway |
| E2E | sob demanda | Fluxos multi-sistema |

## Convenções de infraestrutura

- **`AbstractIntegrationTest`** (base de todos os ITs):
  - Postgres via **singleton container** (start manual no initializer estático, sem `@Container`/`@Testcontainers` — evita parar o container entre classes que compartilham o contexto cacheado).
  - `@ServiceConnection` injeta a conexão real; props dummy satisfazem os placeholders de env.
  - Desliga `spring.docker.compose.enabled` para não conflitar com o compose local.
- **`AuthenticationTestSupport`** ( módulo identity): estende a base com `MockMvc` e helpers
  `createUser`/`login`/`cookieValue`.
- **Nomenclatura**: `*IT` roda no failsafe (`mvn verify`); `*Test` roda no surefire (`mvn test`).
- **Estado compartilhado**: ITs do mesmo contexto compartilham o banco; cada teste cria dados
  próprios com emails únicos (sem `@Transactional` de teste — commits reais validam SQL de verdade).

## ITs do módulo identity

| Classe | Cobre |
|---|---|
| `LoginIT` | login HTTP: cookies (HttpOnly/Secure/SameSite/Path), persistência do hash, 401 em credenciais inválidas, 400 em validação, normalização de email |
| `RefreshTokenIT` | rotação de refresh token (revoga antigo, emite novo), reuso rejeitado, expirado rejeitado, isolamento entre sessões |
| `LogoutIT` | logout revoga sessão atual, cookies expirados (Max-Age=0), idempotência, outras sessões preservadas |
| `CookieAuthenticationIT` | filter chain JWT via cookie: acesso autorizado, 401 sem cookie/garbage/chave desconhecida/token expirado |

## Regras práticas

1. **Bug de integração se testa com IT**: comportamento HTTP, SQL, transação, serialização e
   segurança são testados verticalmente, com Spring e banco reais.
2. **Mock é exceção**: usar `@MockitoBean` apenas para dependências externas lentas/não
   determinísticas (ex.: relógio). Preferir manipular estado real (ex.: UPDATE no `expires_at`).
3. **Teste primeiro, correção depois**: escrever o IT expressando o comportamento correto,
   ver falhar, corrigir a aplicação (foi assim que `LoginIT` expôs o 500 em credenciais inválidas
   e a falta de normalização de email).
4. **Sem banco = unitário**: se o teste não toca Spring/IO e é determinístico, é um unitário puro.
