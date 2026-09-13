package com.marcos.meudinheiro.transaction.infraestructure.repository;

import com.marcos.meudinheiro.transaction.domain.model.CheckInSnapshotModel;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;

import java.time.LocalDate;
import java.util.Optional;
import java.util.UUID;

@Repository
public interface CheckInSnapshotRepository extends JpaRepository<CheckInSnapshotModel, UUID> {

    Optional<CheckInSnapshotModel> findByUserIdAndCheckInDate(UUID userId, LocalDate checkInDate);
}
