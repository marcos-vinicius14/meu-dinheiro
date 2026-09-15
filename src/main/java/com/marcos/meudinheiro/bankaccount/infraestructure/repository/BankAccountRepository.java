package com.marcos.meudinheiro.bankaccount.infraestructure.repository;

import com.marcos.meudinheiro.bankaccount.domain.model.BankAccountModel;
import java.util.List;
import java.util.UUID;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

@Repository
public interface BankAccountRepository extends JpaRepository<BankAccountModel, UUID> {

  List<BankAccountModel> findAllByUserId(UUID userId);

  long countByUserId(UUID userId);

  @Query(
      value = "SELECT EXISTS (SELECT 1 FROM tb_transactions WHERE bank_account_id = :accountId)",
      nativeQuery = true)
  boolean hasTransactions(@Param("accountId") UUID accountId);

  @Query(value = "SELECT pg_advisory_xact_lock(:lockId)", nativeQuery = true)
  void acquireCreationLock(@Param("lockId") long lockId);
}
