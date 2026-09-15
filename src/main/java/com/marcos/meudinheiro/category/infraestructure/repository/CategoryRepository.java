package com.marcos.meudinheiro.category.infraestructure.repository;

import com.marcos.meudinheiro.category.domain.model.CategoryModel;
import java.util.List;
import java.util.UUID;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

@Repository
public interface CategoryRepository extends JpaRepository<CategoryModel, UUID> {

  List<CategoryModel> findAllByUserId(UUID userId);

  boolean existsByUserIdAndDescriptionIgnoreCase(UUID userId, String description);

  @Query(
      """
            SELECT COUNT(c) > 0 FROM CategoryModel c
            WHERE c.user.id = :userId
              AND LOWER(c.description) = LOWER(:description)
              AND c.id <> :categoryId
            """)
  boolean existsByUserIdAndDescriptionExcludingId(
      @Param("userId") UUID userId,
      @Param("description") String description,
      @Param("categoryId") UUID categoryId);

  @Modifying
  @Query(
      value =
          """
            DELETE FROM tb_categories
            WHERE id = :categoryId
              AND NOT EXISTS (SELECT 1 FROM tb_transactions WHERE category_id = :categoryId)
            """,
      nativeQuery = true)
  int deleteIfNoTransactions(@Param("categoryId") UUID categoryId);

  @Query(value = "SELECT pg_advisory_xact_lock(:lockId)", nativeQuery = true)
  void acquireCreationLock(@Param("lockId") long lockId);
}
