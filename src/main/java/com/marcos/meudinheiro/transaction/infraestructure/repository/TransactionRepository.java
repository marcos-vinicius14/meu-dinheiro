package com.marcos.meudinheiro.transaction.infraestructure.repository;

import com.marcos.meudinheiro.transaction.domain.model.TransactionModel;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;

import java.util.List;
import java.util.UUID;

@Repository
public interface TransactionRepository extends JpaRepository<TransactionModel, UUID> {

    List<TransactionModel> findAllByUserIdOrderByDueDateDesc(UUID userId);
}
