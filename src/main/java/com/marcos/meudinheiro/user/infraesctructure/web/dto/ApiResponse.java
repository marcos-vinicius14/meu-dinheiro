package com.marcos.meudinheiro.user.infraesctructure.web.dto;

import java.util.List;

public record ApiResponse(
        List<String> messages
) {
}
