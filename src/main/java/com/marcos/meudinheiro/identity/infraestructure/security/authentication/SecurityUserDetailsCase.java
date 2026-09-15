package com.marcos.meudinheiro.identity.infraestructure.security.authentication;

import com.marcos.meudinheiro.user.application.contract.FindUserIdentityUseCase;
import java.util.List;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.security.core.userdetails.UserDetails;
import org.springframework.security.core.userdetails.UserDetailsService;
import org.springframework.security.core.userdetails.UsernameNotFoundException;
import org.springframework.stereotype.Service;

@Service
public class SecurityUserDetailsCase implements UserDetailsService {

  private static final List<GrantedAuthority> DEFAULT_AUTHORITIES =
      List.of(new SimpleGrantedAuthority("ROLE_USER"));

  private final FindUserIdentityUseCase findUserIdentityUseCase;

  public SecurityUserDetailsCase(FindUserIdentityUseCase findUserIdentityUseCase) {
    this.findUserIdentityUseCase = findUserIdentityUseCase;
  }

  @Override
  public UserDetails loadUserByUsername(String email) {
    var identity =
        findUserIdentityUseCase
            .findByEmail(email)
            .orElseThrow(() -> new UsernameNotFoundException("Invalid credentials"));

    return new AuthenticadedUser(
        identity.id(), identity.email(), identity.passwordHash(), DEFAULT_AUTHORITIES);
  }
}
