# 02-02-SUMMARY: Teclado Persistente 2x2, Consulta do S2S e Lançamento de Gastos

## Visão Geral
Conclusão da segunda onda da Fase 2 (Comandos do Motor Preditivo & Operação Diária), implementando a interface de operação ágil no Bot do Telegram (`apps/bot`):
- Teclado persistente 2x2 (`ReplyKeyboardMarkup`) visível na base do chat com acesso em 1 clique aos 4 comandos centrais do produto (`💰 S2S Hoje`, `💸 Lançar Gasto`, `🔮 Simular`, `📝 Check-in`).
- Consulta do S2S via `/s2s` ou botão `[ 💰 S2S Hoje ]` com exibição do Card Completo de Ciclo.
- Lançamento rápido de despesas via `/gasto <valor> <descrição>` com botões inline de categoria e feedback imediato de impacto no S2S (Antes vs Depois).
- Orientação amigável imediata com exemplos práticos ao acionar `/gasto` sem argumentos ou tocar em `[ 💸 Lançar Gasto ]`.

---

## Entregas Realizadas

### 1. Teclado Persistente 2x2 (`keyboards.go` e `handlers.go`)
- Arquivo: `apps/bot/internal/bot/keyboards.go` e `apps/bot/internal/bot/fsm/keyboards.go`
- `PersistentMenuKeyboard()` implementado com `ResizeKeyboard = true` e layout 2x2:
  - Linha 1: `[ 💰 S2S Hoje ]` e `[ 💸 Lançar Gasto ]`
  - Linha 2: `[ 🔮 Simular ]` e `[ 📝 Check-in ]`
- Ativado automaticamente:
  - No `/start` de usuários já cadastrados (ao apresentar o painel do dia).
  - Na conclusão do onboarding (ao exibir o relatório financeiro inicial).
  - Em todas as respostas informativas do bot para assegurar que o teclado permaneça no rodapé da conversa.

### 2. Card Completo de Ciclo no `/s2s` (`bot.go`)
- Interceptação de `/s2s` e toque em `ButtonS2SToday` (`"💰 S2S Hoje"`).
- Consulta o contexto financeiro consolidado do usuário via `client.GetUserContextByTelegram`.
- Exibe o Card Completo com todos os 5 campos (D-01):
  - Saldo Seguro Diário formatado em pt-BR (ex: `R$ 78,50/dia`).
  - Badge de saúde com emoji padronizado (`🟢 SAUDÁVEL`, `🟡 RESTRITO`, `🔴 RISCO DE DÉFICIT`).
  - Contagem de dias restantes no ciclo e data de término formatada (`DD/MM/AAAA`).
  - Saldo líquido disponível.
  - Patrimônio líquido total.
- Tratamento defensivo para usuários não cadastrados com orientação de `/start`.

### 3. Lançamento Rápido de Gastos com Botões Inline (`bot.go`, `parsers.go`)
- `ParseExpenseCommand` em `fsm/parsers.go`: suporta múltiplos formatos monetários pt-BR (`34.90 Almoço`, `R$ 45,50 Farmácia`, `120 Mercado`, `Almoço executivo 35.00`), valores no início ou fim, e rejeita entradas vazias ou valores negativos.
- Ao enviar `/gasto` sem argumentos ou tocar em `[ 💸 Lançar Gasto ]`: responde na hora com guia de sintaxe e 3 exemplos práticos sem travar a conversa (D-03, D-15).
- Ao enviar `/gasto <valor> <descrição>`:
  - Armazena a despesa pendente em memória temporária segura contra estouro de 64 bytes no callback.
  - Exibe teclado inline com categorias (`Alimentação`, `Transporte`, `Lazer`, `Moradia`, `Saúde`, `Outros`) e botão `[ ❌ Cancelar ]`.
  - No clique da categoria, dispara `QuickExpense` na API e devolve o comparativo Antes vs Depois (D-02):
    - Gasto confirmado com categoria.
    - S2S anterior vs novo S2S.
    - Variação diária negativa destacada (`-R$ X,XX/dia`).
    - Dias restantes e status do ciclo.

---

## Verificação e Qualidade
- `cd apps/bot && go test -v -race ./internal/bot/... -run "TestS2S|TestGasto|TestPersistent"` -> **PASS**
- `cd apps/bot && go test -v -race ./internal/bot/fsm/... -run TestParseExpenseCommand` -> **PASS**
- `make test-bot` -> **PASS** (100% verde com detecção de data races `-race`).
