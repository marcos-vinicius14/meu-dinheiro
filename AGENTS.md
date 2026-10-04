# AGENTS.md

## Comandos essenciais

Monorepo Go + Vue 3.

```bash
make test         # Executa todos os testes (apps/api e apps/bot)
make test-api     # Testes unitários e de integração com Postgres 18 via Testcontainers
make test-bot     # Testes do bot do Telegram
make run-api      # Inicia a API Go localmente (porta 8080)
make run-bot      # Inicia o Bot Telegram em Go
make run-web      # Inicia o servidor de desenvolvimento Vue 3 / Vite (porta 3000)
make build-all    # Compila os binários de api, bot e gera o build de produção do web
make up           # Sobe o banco PostgreSQL 18 local via Docker Compose (compose.yaml / .env.local)
make down         # Para os containers de desenvolvimento
make up-prd       # Sobe a stack completa de produção (compose.prd.yaml / .env.prd)
make down-prd     # Para os containers de produção
make release TAG=v0.1.0  # Cria a tag Git anotada, publica no remoto e aciona deploy no Coolify
```

- Docker é obrigatório para rodar os testes de integração (`Testcontainers` sobe `postgres:18-alpine` automaticamente).
- **Ambientes**:
  - **Local/Dev**: `compose.yaml` (apenas PostgreSQL 18) e variáveis em `.env.local` (com fallback para `.env`).
  - **Produção**: `compose.prd.yaml` (stack completa: PostgreSQL 18, API e Bot) e variáveis em `.env.prd`.
- Verificação completa: `make test` e `make build-all`.

---

## Arquitetura: Monorepo

O projeto está organizado no diretório `apps/`:

- **`apps/api`**: API RESTful escrita em **Go** (idiomática e direta).
  - Roteamento: `go-chi/chi/v5`
  - Banco de Dados: `jackc/pgx/v5` com connection pool (`pgxpool`)
  - Cálculos Financeiros: `shopspring/decimal` (arredondamento bancário HalfEven, precisão de 2 casas, divisão segura)
  - Migrações: `pressly/goose/v3` com migrations SQL embutidas (`embed.FS`)
  - Autenticação: Telegram-first (`telegram_id` único, desafio deep link, cookies de sessão JWT HTTP-only)
  - Layout de pacotes:
    - `cmd/api`: Ponto de entrada e graceful shutdown.
    - `internal/auth`: Desafios de autenticação Telegram, middleware de proteção e emissão de JWT.
    - `internal/bankaccount`: CRUD de contas com locking transacional e limite defensivo de 3 contas.
    - `internal/category`: CRUD de categorias com locking e verificação case-insensitive de unicidade.
    - `internal/transaction`: Transações avulsas, pacotes parcelados (imutabilidade de parcelas), check-in diário atômico e simulador what-if.
    - `internal/transaction/engine`: Motor de cálculo determinístico de S2S (Saldo Seguro Diário) e simulador multi-ciclo de 12 meses.
    - `internal/user`: Gestão de usuários vinculados ao Telegram com seeding automático de categorias e conta inicial.
    - `internal/web`: Roteador HTTP, middlewares de logging/recovery, helpers JSON e tratamento de cookies.
    - `tests/integration`: Suíte de testes ponta a ponta contra PostgreSQL 18 real via Testcontainers.
- **`apps/bot`**: Bot para Telegram escrito em **Go** (`go-telegram-bot-api/telegram-bot-api/v5`).
  - Processa `/start auth_<token>` para autorizar o login web em 1 clique.
  - Provê comandos de suporte e consulta de status.
- **`apps/web`**: Aplicação e Landing Page construída com **Vue 3** e **Vite**.
  - Apresentação do produto (S2S diário, simulador de 12 meses).
  - Fluxo de login 1-click com Telegram via deep link e polling automático com cookies de sessão.
  - Prévia do painel financeiro para usuários autenticados.

---

## Autenticação Telegram-First

A autenticação legada (email, senha, BCrypt, chaves RSA PEM, tokens de refresh) foi **totalmente substituída** pelo modelo nativo via Telegram:
1. O usuário no navegador clica em "Entrar com Telegram".
2. O front-end chama `POST /auth/telegram/challenge` e recebe um token de desafio criptográfico efêmero (TTL de 5 minutos) com deep link `tg://resolve?domain=meu_dinheiro_bot&start=auth_<token>`.
3. O front-end inicia polling periódico em `GET /auth/telegram/poll?token=<token>`.
4. O usuário clica no link e inicia a conversa no Telegram. O Bot processa o token e chama o endpoint interno `POST /internal/auth/authorize-challenge`.
5. A API localiza ou cria o usuário pelo `telegram_id` e marca o desafio como autorizado.
6. A próxima chamada de polling do navegador recebe status `authorized` e um cookie `session_token` HTTP-only seguro contendo o JWT de sessão.
7. O usuário está autenticado e pronto para operar.

---

## Diretrizes de Engenharia e Boas Práticas

1. **Mensagens em Português (pt-BR)**:
   - Erros de validação e respostas da API devem ser claras e em português (ex.: `"categoria não encontrada"`, `"limite máximo de 3 contas atingido"`, `"saldo flexível insuficiente"`).
2. **Precisão Financeira**:
   - Nunca use float para valores monetários. Use sempre `decimal.Decimal` com scale 2 e arredondamento `RoundBank` (HalfEven).
3. **Locking e Concorrência**:
   - Modificações em listas com limites de quantidade (contas) e regras de unicidade sob alto paralelismo usam `pg_advisory_xact_lock` no PostgreSQL para garantir consistência absoluta sem deadlock.
4. **Testes (Troféu de Testes)**:
   - Testes unitários para lógica pura e invariantes matemáticas (`money_test.go`, `date_interval_test.go`, `predictive_engine_test.go`, `what_if_simulator_test.go`).
   - Testes de integração para todos os endpoints REST contra banco PostgreSQL 18 real com Testcontainers. Proibido usar mocks de banco.
5. **Modularidade, Responsabilidade Única & Legibilidade**:
   - **Proibido "God Files"**: Arquivos grandes acumulando múltiplas responsabilidades são expressamente proibidos. Mantenha os arquivos coesos e bem delimitados (ex.: `types.go`, `parsers.go`, `keyboards.go`, `store.go`, `handlers.go`).
   - **Responsabilidades Bem Definidas (SRP)**: Cada função, struct e método deve ter um propósito único e claro. Se um switch ou fluxo crescer além do razoável, decomponha em métodos ou handlers dedicados.
   - **Facilidade de Manutenção & Extensibilidade**: O código deve ser aberto para extensão e fechado para modificação (Open/Closed). Use tipagem forte e enums/constantes em vez de strings literais soltas (`type Action string`).
   - **Legibilidade & Nomenclatura Expressiva**: Nunca economize no nome de variáveis, parâmetros ou funções. É proibido usar variáveis crípticas de 1 ou 2 letras para entidades de domínio (use `userFinancialContext`, `monthlyEssentialCost`, `currentSession` em vez de `u`, `c`, `s`). Variáveis devem ser autodocumentadas e expressar claramente a regra de negócio.

---

## Git / PR

- **Autorização Formal Obrigatória**: É **terminantemente proibido** executar `git commit`, `git push`, criação ou merge de branches/PRs sem confirmação expressa do usuário.
- Padrão Conventional Commits (`feat:`, `fix:`, `chore:`, `refactor:`).
- Nunca commitar arquivos `.env` ou credenciais.
