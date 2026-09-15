package com.marcos.meudinheiro.transaction.infraestructure.repository;

import com.marcos.meudinheiro.transaction.domain.model.TransactionBundleModel;
import java.util.UUID;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;

@Repository
public interface TransactionBundleRepository extends JpaRepository<TransactionBundleModel, UUID> {}
