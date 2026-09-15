package com.marcos.meudinheiro.user.application.usecase;

public class CreateUserUseCaseIT {
  // TODO: Escrever testes com o seguinte cenario:
  /*
  create user válido
      → 204
      → registro salvo no Postgres

  email inválido
      → 400
      → não persiste

  email duplicado
      → 409
      → não cria segundo usuário

  nome inválido
      → 400

  senha inválida
      → 400

  UUID
      → gerado pelo PostgreSQL
      → persistido corretamente
  */

}
