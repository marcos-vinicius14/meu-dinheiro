package com.marcos.meudinheiro.transaction.infraestructure.repository;

import com.marcos.meudinheiro.transaction.domain.model.TransactionBundleModel;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;

import java.util.UUID;

@Repository
public interface TransactionBundleRepository extends JpaRepository<TransactionBundleModel, UUID> {
}
