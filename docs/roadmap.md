v1  Modular Monolith + Hexagonal
v2  Ledger de dupla entrada
v3  Domain Events
v4  Importação OFX/CSV
v5  Idempotência + deduplicação
v6  Conciliação automática
v7  Outbox + RabbitMQ
v8  CQRS para relatórios
v9  Recorrências e projeções
v10 Investimentos
v11 Open Finance
v12 Event Sourcing em alguns agregados
v13 Observabilidade completa
v14 Separação seletiva em serviços


### Description
Ledger contábil de dupla entrada para garantir consistência.
Importação bancária via OFX/CSV/Open Finance.
Conciliação automática entre lançamentos importados e cadastrados.
Categorização inteligente baseada em regras e histórico.
Parcelamentos, recorrências e previsões.
Orçamentos mensais com projeções.
Investimentos e patrimônio.
Eventos de domínio para movimentações financeiras.
Outbox + mensageria para processamento assíncrono.
CQRS para relatórios e dashboards.
Snapshots para acelerar consultas históricas.
Event Sourcing em contas ou transações, se quiser levar ao extremo.
Idempotência para importações bancárias.
Locks e concorrência para evitar saldo inconsistente.
Auditoria imutável.
Multi-currency e conversão cambial.
Motor de regras para categorização e alertas.
Projeção de fluxo de caixa.
Detecção de anomalias em gastos.
Notificações orientadas a eventos.
OpenTelemetry + métricas + tracing.
Testcontainers, contract tests e mutation testing.