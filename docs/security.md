# Decisões de Segurança — Módulo identity

Documento de decisões (ADR informal). Cada item registra o "porquê" que não aparece no código.

## Modelo de sessão

| Token | Tipo | TTL | Storage |
|---|---|---|---|
| `access_token` | JWT RS256 (issuer `meudinheiro`) | 10 min | Cookie httpOnly/Secure/SameSite=Strict, path `/` |
| `refresh_token` | Opaque aleatório (hash SHA-256 no banco) | 15 dias | Cookie httpOnly/Secure/SameSite=Strict, path `/auth` |

Rotação de refresh a cada uso: token antigo é revogado e um novo é emitido.

## 1. Sem denylist para access token (decisão)

**Decisão: não implementar revogação imediata do access token.**

- JWT é stateless por design; denylist exigiria consulta (DB ou cache) a cada request no
  resource server — estado onde não deveria existir.
- Janela de exposição pós-logout = TTL do access (≤ 10 min), aceitável para o v1.
- Mitigações existentes: TTL curto + rotação de refresh + detecção de reuso.
- Revisar se surgir requisito de compliance que exija revogação imediata.

## 2. Detecção de reuso de refresh token (roubo presumido)

Refresh token **revogado** apresentado novamente = indício de roubo (o legítimo já rotacionou).

- Ação: revogar **todos** os refresh tokens do usuário (`revokeAllByUserId`).
- Colateral aceito: derruba sessões legítimas de outros dispositivos; usuário reloga.
- Token **expirado** por idade não dispara detecção (expiração é evento normal, não roubo).
- Alternativa futura: famílias de tokens (family_id) para revogar só a cadeia comprometida
  — exige migration; anotado no roadmap de melhorias.

## 3. Rate limit por IP no endpoint de login

Camada complementar ao lockout: throttle de **volume de requisições por IP de origem** em
`POST /auth/login` (`LoginRateLimitFilter`, janela fixa via Caffeine).

| | Lockout (item 2) | Rate limit por IP |
|---|---|---|
| Chave | email | IP de origem (`remoteAddr`) |
| Conta | tentativas **falhas** | todas as requisições (válidas ou não) |
| Ação ao exceder | 401 opaco (indistinguível de senha errada) | **429** + mensagem |
| Config | `security.login.max-attempts` (5), `security.login.lockout` (15m) | `security.login.rate-limit.max-requests` (10), `security.login.rate-limit.window` (1m) |

- 429 é visível (não vaza estado de credencial — IP não é segredo; distinto do lockout, que é opaco por design).
- Conta requisições válidas também: o objetivo é limitar volume/CPU, não apenas falhas.
- Janela fixa: contador reseta ao expirar a janela (`CaffeineLoginRateLimiterTest` cobre com Clock mutável).
- **IP real em produção**: usa `request.getRemoteAddr()`. Atrás de proxy/LB isso é o IP do proxy —
  configurar `server.forward-headers-strategy` quando houver proxy confiável; NÃO confiar em
  `X-Forwarded-For` sem proxy confiável (spoofável → limpeza de cache por atacante).
- In-memory por instância (mesma limitação do lockout).

## 4. Lockout do login (Caffeine, in-memory)

- Após N falhas (`security.login.max-attempts`, default 5), bloqueia por
  `security.login.lockout` (default 15m) — caches Caffeine com `expireAfterWrite`.
- **Mensagem opaca**: bloqueio devolve a mesma resposta de credencial inválida (401 +
  "Credenciais inválidas"). Não revela que a conta existe nem permite enumeração.
- Contador reseta em login com sucesso.
- In-memory ⇒ por instância. Adequado para monólito single-instance v1; migrar para
  Redis/Bucket4j distribuído quando escalar horizontalmente.

## 5. Cleanup de tokens

Job `@Scheduled` (03:00 diário) remove tokens com `expires_at` ou `revoked_at` anteriores a
`now - security.refresh-token.retention` (default 7d). Retenção > 0 preserva tokens
revogados/expirados recentes para suporte à detecção de reuso e auditoria.

## Endpoints

| Endpoint | Auth | Comportamento |
|---|---|---|
| `POST /auth/login` | público | 204 + cookies; 401 opaco (credencial errada ou bloqueado) |
| `POST /auth/refresh` | público (cookie refresh é a credencial) | 204 + cookies novos; 401 + cookies limpos |
| `POST /auth/logout` | público (idempotente) | revoga sessão do cookie; sempre 204 |
| `POST /auth/logout-all` | **autenticado** (access token) | revoga todas as sessões do usuário |
| `GET /auth/me` | autenticado | dados do usuário corrente |

## Mensagens opacas — regra geral

Nunca confirme existência/estado de credenciais: refresh inválido, expirado e revogado são
indistinguíveis na resposta ("Refresh token inválido, expirado ou revogado"); login bloqueado
é indistinguível de senha errada. Ver também `docs/notification-pattern.md`.
