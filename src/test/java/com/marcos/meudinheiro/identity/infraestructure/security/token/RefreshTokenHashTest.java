package com.marcos.meudinheiro.identity.infraestructure.security.token;

import static org.assertj.core.api.Assertions.assertThat;

import org.junit.jupiter.api.Test;

class RefreshTokenHashTest {

  @Test
  void hashesKnownVector() {
    var hash = RefreshTokenHash.sha256("hello");

    assertThat(hash)
        .isEqualTo("2cf24dba5fb0a30e26e83b2ac5b9e29e" + "1b161e5c1fa7425e73043362938b9824");
  }

  @Test
  void hashesEmptyString() {
    var hash = RefreshTokenHash.sha256("");

    assertThat(hash)
        .isEqualTo("e3b0c44298fc1c149afbf4c8996fb924" + "27ae41e4649b934ca495991b7852b855");
  }

  @Test
  void produces64LowercaseHexCharacters() {
    var hash = RefreshTokenHash.sha256("qualquer-token-opaco");

    assertThat(hash).hasSize(64);
    assertThat(hash).matches("[0-9a-f]{64}");
  }

  @Test
  void sameInputProducesSameHash() {
    assertThat(RefreshTokenHash.sha256("token")).isEqualTo(RefreshTokenHash.sha256("token"));
  }

  @Test
  void differentInputsProduceDifferentHashes() {
    assertThat(RefreshTokenHash.sha256("token-1")).isNotEqualTo(RefreshTokenHash.sha256("token-2"));
  }
}
