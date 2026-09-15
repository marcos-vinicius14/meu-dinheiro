package com.marcos.meudinheiro.identity.application.contract.dto;

public record AuthenticationOutput(
    String acessToken,
    String tokenType,
    long expiresIn,
    String refreshToken,
    long refreshTokenExpiresIn) {
  public static AuthenticationOutput bearer(
      String acessToken, long expiresIn, String refreshToken, long refreshTokenExpiresIn) {
    return new AuthenticationOutput(
        acessToken, "Bearer", expiresIn, refreshToken, refreshTokenExpiresIn);
  }
}
