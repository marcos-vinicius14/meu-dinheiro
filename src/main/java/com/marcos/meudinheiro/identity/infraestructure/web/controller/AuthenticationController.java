package com.marcos.meudinheiro.identity.infraestructure.web.controller;

import com.marcos.meudinheiro.identity.application.contract.AuthenticateUserUseCase;
import com.marcos.meudinheiro.identity.application.contract.CurrentIdentity;
import com.marcos.meudinheiro.identity.application.contract.LogoutAllUseCase;
import com.marcos.meudinheiro.identity.application.contract.LogoutUseCase;
import com.marcos.meudinheiro.identity.application.contract.RefreshTokenUseCase;
import com.marcos.meudinheiro.identity.application.contract.dto.AuthenticationInput;
import com.marcos.meudinheiro.identity.application.contract.dto.AuthenticationOutput;
import com.marcos.meudinheiro.identity.infraestructure.web.dto.CurrentUserResponse;
import com.marcos.meudinheiro.identity.infraestructure.web.dto.LoginRequest;
import com.marcos.meudinheiro.shared.infraestructure.web.response.ErrorResponse;
import com.marcos.meudinheiro.user.application.contract.FindUserIdentityUseCase;
import jakarta.persistence.EntityNotFoundException;
import jakarta.servlet.http.HttpServletResponse;
import jakarta.validation.Valid;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseCookie;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.CookieValue;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/auth")
final class AuthenticationController {
  private final AuthenticateUserUseCase useCase;
  private final LogoutUseCase logoutUseCase;
  private final LogoutAllUseCase logoutAllUseCase;
  private final RefreshTokenUseCase refreshTokenUseCase;
  private final CurrentIdentity currentIdentity;
  private final FindUserIdentityUseCase findUserIdentity;

  AuthenticationController(
      AuthenticateUserUseCase useCase,
      LogoutUseCase logoutUseCase,
      LogoutAllUseCase logoutAllUseCase,
      RefreshTokenUseCase refreshTokenUseCase,
      CurrentIdentity currentIdentity,
      FindUserIdentityUseCase findUserIdentity) {
    this.useCase = useCase;
    this.logoutUseCase = logoutUseCase;
    this.logoutAllUseCase = logoutAllUseCase;
    this.refreshTokenUseCase = refreshTokenUseCase;
    this.currentIdentity = currentIdentity;
    this.findUserIdentity = findUserIdentity;
  }

  @PostMapping("/login")
  ResponseEntity<ErrorResponse> login(
      @Valid @RequestBody LoginRequest request, HttpServletResponse response) {

    // TODO: Mover para um mapper
    var input = new AuthenticationInput(request.email(), request.password());

    var result = useCase.execute(input);

    if (result.isFailure()) {
      return ResponseEntity.status(HttpStatus.UNAUTHORIZED)
          .body(new ErrorResponse(result.errors()));
    }

    var authentication = result.value();

    response.addHeader(HttpHeaders.SET_COOKIE, accessTokenCookie(authentication).toString());

    response.addHeader(HttpHeaders.SET_COOKIE, refreshTokenCookie(authentication).toString());

    return ResponseEntity.noContent().build();
  }

  @PostMapping("/logout")
  ResponseEntity<Void> logout(
      @CookieValue(name = "refresh_token", required = false) String refreshToken,
      HttpServletResponse response) {
    if (refreshToken != null) {
      logoutUseCase.execute(refreshToken);
    }

    response.addHeader(HttpHeaders.SET_COOKIE, expiredAccessTokenCookie().toString());

    response.addHeader(HttpHeaders.SET_COOKIE, expiredRefreshTokenCookie().toString());

    return ResponseEntity.noContent().build();
  }

  @GetMapping("/me")
  ResponseEntity<CurrentUserResponse> me() {
    var userId = currentIdentity.findCurrentAuthenticadedUser();

    var identity = findUserIdentity.findById(userId).orElseThrow(EntityNotFoundException::new);

    return ResponseEntity.ok(new CurrentUserResponse(identity.id(), identity.email()));
  }

  @PostMapping("/logout-all")
  ResponseEntity<Void> logoutAll(HttpServletResponse response) {
    logoutAllUseCase.execute();

    expireAuthCookies(response);

    return ResponseEntity.noContent().build();
  }

  @PostMapping("/refresh")
  ResponseEntity<ErrorResponse> refresh(
      @CookieValue(name = "refresh_token", required = false) String refreshToken,
      HttpServletResponse response) {
    var result = refreshTokenUseCase.execute(refreshToken);

    if (result.isFailure()) {
      expireAuthCookies(response);

      return ResponseEntity.status(HttpStatus.UNAUTHORIZED)
          .body(new ErrorResponse(result.errors()));
    }

    var authentication = result.value();

    response.addHeader(HttpHeaders.SET_COOKIE, accessTokenCookie(authentication).toString());

    response.addHeader(HttpHeaders.SET_COOKIE, refreshTokenCookie(authentication).toString());

    return ResponseEntity.noContent().build();
  }

  private void expireAuthCookies(HttpServletResponse response) {
    response.addHeader(HttpHeaders.SET_COOKIE, expiredAccessTokenCookie().toString());

    response.addHeader(HttpHeaders.SET_COOKIE, expiredRefreshTokenCookie().toString());
  }

  private ResponseCookie accessTokenCookie(AuthenticationOutput authentication) {
    return ResponseCookie.from("access_token", authentication.acessToken())
        .httpOnly(true)
        .secure(true)
        .sameSite("Strict")
        .path("/")
        .maxAge(authentication.expiresIn())
        .build();
  }

  private ResponseCookie refreshTokenCookie(AuthenticationOutput authentication) {
    return ResponseCookie.from("refresh_token", authentication.refreshToken())
        .httpOnly(true)
        .secure(true)
        .sameSite("Strict")
        .path("/auth")
        .maxAge(authentication.refreshTokenExpiresIn())
        .build();
  }

  private ResponseCookie expiredAccessTokenCookie() {
    return ResponseCookie.from("access_token", "")
        .httpOnly(true)
        .secure(true)
        .sameSite("Strict")
        .path("/")
        .maxAge(0)
        .build();
  }

  private ResponseCookie expiredRefreshTokenCookie() {
    return ResponseCookie.from("refresh_token", "")
        .httpOnly(true)
        .secure(true)
        .sameSite("Strict")
        .path("/auth")
        .maxAge(0)
        .build();
  }
}
