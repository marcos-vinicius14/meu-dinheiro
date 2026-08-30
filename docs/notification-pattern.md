# Notification Pattern

Regra central do projeto: **exceção é apenas para casos excepcionais**. Violação de regra de
negócio é um resultado *esperado* da execução — retorna como valor, não como exceção.

```
Exceção  = "não era para acontecer"  → infraestrutura caiu, bug, contrato de framework
Resultado = "era uma possibilidade"   → senha errada, token expirado, email duplicado
```

## Componentes (`shared/notification`)

| Tipo | Papel | Onde vive |
|---|---|---|
| `ValidationResult<T>` | UMA validação: ou o valor válido (`value`), ou erros | Value objects e validações de domínio (ex.: `Email.create`) |
| `Notification` | Acumula erros de várias validações (`collect`) | Dentro do use case, durante a execução |
| `OperationResult<T>` | Resultado final do use case: ou o payload (`value`), ou `errors` | Retorno de use cases (porta da aplicação) |

```java
// ValidationResult — uma validação isolada
ValidationResult.valid(valor)          // value presente, errors vazio
ValidationResult.invalid("Email inválido")

// Notification — coletor de erros do use case
Notification notification = new Notification();
UserModel user = UserMapper.toUserModel(request, notification); // coleta erros internamente
notification.collect(ValidationResult<T>)                        // coleta e devolve value (ou null)

// OperationResult<T> — saída do use case
OperationResult.success(payload)   // use case com retorno
OperationResult.success()          // use case void (OperationResult<Void>)
OperationResult.failure(errors)    // List<String> ou String única
```

Invariantes: `isSuccess()` ⇔ `errors` vazio ⇔ `value` presente. Em `failure`, `value()` é `null`.

## Fluxo padrão de um use case

1. Cria o `Notification`.
2. Valida/resolve cada pré-condição, coletando no notification
   (`collect(ValidationResult)`, `ValidationResult.invalid(...)` para falhas diretas).
3. Ao detectar `notification.hasErrors()`, retorna `OperationResult.failure(notification.errors())`.
4. Caminho feliz executa e retorna `OperationResult.success(payload)`.

### Exemplo real — `RefreshTokenUseCaseImpl`

```java
var notification = new Notification();

var refreshToken = notification.collect(
        resolveRefreshToken(rawRefreshToken, now));   // ValidationResult<RefreshTokenModel>

if (notification.hasErrors()) {
    return OperationResult.failure(notification.errors());
}

refreshToken.revoke(now);

var identity = notification.collect(
        resolveIdentity(refreshToken.getUserId()));   // ValidationResult<UserIdentityOutput>

if (notification.hasErrors()) {
    return OperationResult.failure(notification.errors());
}

// ... caminho feliz ...
return OperationResult.success(AuthenticationOutput.bearer(...));
```

## Mapeamento para HTTP (controller)

O controller decide o status; o use case não conhece HTTP:

| Resultado | Status típico |
|---|---|
| `isSuccess()` | 200/204 (payload no corpo ou cookies) |
| `isFailure()` com erro de entrada/regra | 400/409/401/422 — depende do caso |

Referência: `UserController.create` → falha vira 400 com `ErrorResponse(errors)`.

## Quando usar exceção (excepcional de verdade)

| Situação | Tratamento |
|---|---|
| Banco fora do ar, timeout, falha de infraestrutura | Propaga; `GlobalExceptionHandler` → 500 |
| Contrato de framework: `BadCredentialsException` do Spring Security no login | Propaga; handler de `AuthenticationException` → 401 |
| Invariante interna quebrada (bug) | Propaga; nunca captura para "tratar" |
| Regra de negócio violada (input do usuário, estado válido do domínio) | **Notification + OperationResult, nunca exceção** |

Princípio prático: se o caller precisa distinguir esse desfecho no fluxo normal
("senha errada", "token expirado", "email duplicado"), ele é um *resultado*, não uma exceção.

## Semântica transacional (importante)

Em use cases `@Transactional`:

- `OperationResult.failure(...)` **retorna** → a transação faz **commit**
  (o que já foi alterado até ali é persistido).
- Exceção propagada → **rollback**.

No `RefreshTokenUseCaseImpl` isso é intencional: quando o token é válido mas o usuário não
existe mais, o `revoke(now)` já aplicado **persiste** (limpeza de token órfão é desejável).
Se um use case precisar desfazer tudo ao falhar, valide tudo antes de mutar o agregado.

## Mensagens de erro

- Texto em português, voltado ao consumidor da API (mesmo estilo de `Email.create`).
- Não vaze estado interno: o refresh usa mensagem única opaca
  ("Refresh token inválido, expirado ou revogado") para não revelar se o token existe,
  expirou ou foi revogado.
- Erros de VO são específicos e acionáveis ("Email inválido", "A senha deve ter no mínimo 8 caracteres").
