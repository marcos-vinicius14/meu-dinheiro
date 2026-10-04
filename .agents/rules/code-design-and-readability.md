# Diretrizes de Design de Código, Modularidade e Legibilidade

> **Objetivo:** Garantir que o código do projeto seja limpo, extensível, de fácil manutenção e altamente legível para qualquer desenvolvedor ou agente de IA.

---

## 1. Responsabilidade Única & Proibição de "God Files"

- **Sem Arquivos Monolíticos:** É proibido criar arquivos grandes acumulando múltiplas responsabilidades (ex.: parsing, regras de negócio, chamadas HTTP e renderização de mensagens/teclados no mesmo arquivo).
- **Separação Coesa de Arquivos:** Dentro de um pacote, separe as responsabilidades em arquivos com propósito claro:
  - `types.go`: structs de dados, enums e constantes de domínio.
  - `parsers.go`: rotinas de validação, sanitização e extração de inputs.
  - `keyboards.go` (no bot): geradores de teclado inline e componentes de interação.
  - `handlers.go`: despachantes e orquestração de fluxo.
  - `store.go`: persistência ou gerenciamento de sessão/cache.
- **Funções Curtas e Focadas:** Métodos e funções devem fazer apenas uma coisa e fazê-la bem. Switches extensos devem despachar para métodos especializados privados (ex.: `handleStepBalance`, `handleStepCycleDay`).

---

## 2. Legibilidade & Nomenclatura Expressiva

- **Não Economize no Nome de Variáveis:** O código deve ser autodocumentado.
  - ❌ **Proibido:** variáveis crípticas de 1 ou 2 letras para entidades de domínio (ex: `u`, `c`, `s`, `p`, `req`, `r` para structs ou dados complexos).
  - ✅ **Obrigatório:** nomes descritivos que transmitam o conceito de domínio (ex.: `userFinancialContext`, `monthlyEssentialCost`, `currentSession`, `cycleStartDay`, `emergencyFundMonths`).
  - *Exceção:* variáveis de iteração puramente numéricas em loops simples (`i`, `j`).
- **Nomes em Inglês Técnico para Símbolos / Mensagens em Português:**
  - Nomes de funções, tipos, structs, métodos e variáveis são em inglês técnico idiomático (`CalculateDailySafeBalance`, `FixedExpenseData`).
  - Textos de mensagens voltadas ao usuário, respostas de erro da API e logs de auditoria de negócio são estritamente em **pt-BR**.

---

## 3. Extensibilidade & Facilidade de Manutenção (Open/Closed Principle)

- **Tipagem Forte sobre "Magic Strings":** Sempre crie tipos e constantes (enums) para estados, ações, callbacks e identificadores (ex.: `type Action string`, `ActionFinishFixedExpenses`). Evite comparar strings literais soltas no código.
- **Transições Seguras:** Máquinas de estado e handlers devem verificar o estado atual (`sess.CurrentState`) antes de aceitar ações ou callbacks, evitando ataques de replay e transições ilegais.
- **Desacoplamento via Interfaces:** Use interfaces curtas para dependências externas (`TelegramSender`, `APIClient`, `SessionStore`), viabilizando testes unitários rápidos e determinísticos com fakes sem necessidade de I/O de rede ou banco de dados.
