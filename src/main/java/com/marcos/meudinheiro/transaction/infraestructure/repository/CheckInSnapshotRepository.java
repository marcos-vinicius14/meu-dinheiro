package com.marcos.meudinheiro.transaction.infraestructure.repository;

import com.marcos.meudinheiro.transaction.domain.model.CheckInSnapshotModel;
import java.time.LocalDate;
import java.util.Optional;
import java.util.UUID;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;

@Repository
public interface CheckInSnapshotRepository extends JpaRepository<CheckInSnapshotModel, UUID> {

  Optional<CheckInSnapshotModel> findByUserIdAndCheckInDate(UUID userId, LocalDate checkInDate);
}
