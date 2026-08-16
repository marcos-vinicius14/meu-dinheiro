package com.marcos.meudinheiro.shared.infraestructure.web.response;

import java.util.List;

public record ErrorResponse(
        List<String> errors
) {
}
