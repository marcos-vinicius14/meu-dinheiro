package com.marcos.meudinheiro.identity.application.contract;

import java.util.UUID;

public interface CurrentIdentity {
    UUID findCurrentAuthenticadedUser();
}
