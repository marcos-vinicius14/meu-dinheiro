# Meu Dinheiro

Monorepo de gestão financeira pessoal orientada ao cálculo contínuo de **Saldo Seguro Diário (S2S)** e simulação preditiva de compras futuras (what-if em até 12 ciclos).

Composto por:
- **`apps/api`**: API RESTful de alta performance em Go (Chi + PostgreSQL 18 + Testcontainers).
- **`apps/bot`**: Bot de Telegram para acesso rápido e autenticação sem senha (1-Click Login).
- **`apps/web`**: Landing Page moderna e painel inicial construído em Vue 3 + Vite.

---

## 🚀 Como Iniciar

### Pré-requisitos
- **Go 1.24+**
- **Docker** (para PostgreSQL e Testcontainers)
- **Node.js 20+** e **npm**

### Comandos Rápidos

```bash
# Executar todos os testes automatizados
make test

# Iniciar o banco PostgreSQL de desenvolvimento localmente
make up        # usa compose.yaml e .env.local
make down

# Iniciar a stack completa em modo produção (containers)
make up-prd    # usa compose.prd.yaml e .env.prd
make down-prd

# Iniciar os serviços localmente (desenvolvimento)
make run-api   # http://localhost:8080
make run-bot   # Polling Telegram
make run-web   # http://localhost:3000

# Compilar todos os pacotes para produção
make build-all
```

### Configuração de Ambientes (.env)

- **Local/Dev**: Crie `.env.local` a partir de `.env.example` (`cp .env.example .env.local`). O compose `compose.yaml` gerencia o PostgreSQL local.
- **Produção**: Crie `.env.prd` a partir de `.env.example` (`cp .env.example .env.prd`). O compose `compose.prd.yaml` gerencia a stack completa em containers (PostgreSQL, API e Bot).


---

## 📐 Arquitetura

```mermaid
graph LR
    User[Usuário] -->|Navegador| Web[apps/web (Vue 3)]
    User -->|Telegram App| Bot[apps/bot (Go)]
    Web -->|HTTP / REST| API[apps/api (Go Chi)]
    Bot -->|Internal Auth / HTTP| API
    API -->|pgxpool| DB[(PostgreSQL 18)]
```

### Funcionalidades do Núcleo

1. **Saldo Seguro Diário (S2S)**:
   Calcula deterministicamente o valor que o usuário pode gastar por dia sem entrar em déficit ou consumir a poupança meta (`(Liquidez - ContasFixas - PoupancaMeta) / DiasRestantes`).
2. **Simulador What-If**:
   Permite projetar o impacto de compras à vista ou parceladas (até 12 meses) nos ciclos futuros, identificando gargalos e transições de status (`SAUDÁVEL`, `RESTRITO`, `RISCO DE DÉFICIT`).
3. **Autenticação Nativa por Telegram**:
   Zero senhas no banco. Login no navegador via desafio criptográfico temporário e autorização instantânea no bot.
4. **Precisão e Concorrência**:
   Arredondamento bancário Half-Even (`shopspring/decimal`), locking atômico via `pg_advisory_xact_lock` no PostgreSQL para integridade sob condições de corrida.

---

## 🧪 Testes

O projeto segue a estratégia do **Testing Trophy**:
- **Testes Unitários**: Testam invariantes puras (cálculo de frações monetárias, intervalos de datas, matriz preditiva do S2S).
- **Testes de Integração**: Testam todos os fluxos de ponta a ponta contra uma instância real de **PostgreSQL 18 em container** via `testcontainers-go`.
