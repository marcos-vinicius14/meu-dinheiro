package com.marcos.meudinheiro.transaction.domain.model;

import com.marcos.meudinheiro.shared.valueobjects.Money;
import jakarta.persistence.*;
import java.time.LocalDate;
import java.util.ArrayList;
import java.util.List;
import java.util.UUID;
import org.hibernate.annotations.Generated;
import org.jspecify.annotations.Nullable;

@Entity
@Table(name = "tb_transaction_bundles")
public class TransactionBundleModel {

  @Id
  @Generated
  @Column(name = "id", insertable = false, updatable = false)
  private UUID id;

  @Column(nullable = false, length = 255)
  private String description;

  @Embedded
  @AttributeOverride(
      name = "value",
      column = @Column(name = "total_amount", nullable = false, precision = 19, scale = 2))
  private Money totalAmount;

  @Column(name = "total_installments", nullable = false)
  private Integer totalInstallments;

  @Column(name = "first_due_date", nullable = false)
  private LocalDate firstDueDate;

  @ManyToOne(fetch = FetchType.LAZY, optional = false)
  @JoinColumn(name = "user_id", nullable = false)
  private com.marcos.meudinheiro.user.domain.model.UserModel user;

  protected TransactionBundleModel() {}

  private TransactionBundleModel(
      String description,
      @Nullable Money totalAmount,
      Integer totalInstallments,
      LocalDate firstDueDate,
      com.marcos.meudinheiro.user.domain.model.UserModel user) {
    this.description = description;
    this.totalAmount = totalAmount;
    this.totalInstallments = totalInstallments;
    this.firstDueDate = firstDueDate;
    this.user = user;
  }

  public static TransactionBundleModel create(
      String description,
      @Nullable Money totalAmount,
      int totalInstallments,
      LocalDate firstDueDate,
      com.marcos.meudinheiro.user.domain.model.UserModel user) {
    if (totalAmount == null || totalAmount.isNegative() || totalAmount.isZero()) {
      throw new IllegalArgumentException("Valor total do parcelamento deve ser positivo");
    }
    if (totalInstallments <= 0) {
      throw new IllegalArgumentException("Número de parcelas deve ser positivo");
    }
    if (firstDueDate == null) {
      throw new IllegalArgumentException("Data do primeiro vencimento é obrigatória");
    }
    return new TransactionBundleModel(
        description, totalAmount, totalInstallments, firstDueDate, user);
  }

  /**
   * Gera as N parcelas filhas (COMMITTED) com alocação HALF_EVEN: a primeira parcela absorve o
   * resto de arredondamento, garantindo soma exata do total. Vencimentos mensais a partir de
   * firstDueDate; dia clampado ao fim do mês. bundleId é o id do bundle já persistido;
   * user/category são propagados.
   */
  public List<TransactionModel> generateInstallments(
      UUID bundleId,
      com.marcos.meudinheiro.user.domain.model.UserModel user,
      com.marcos.meudinheiro.category.domain.model.CategoryModel category) {
    if (bundleId == null) {
      throw new IllegalArgumentException("Bundle deve ser persistido antes de gerar parcelas");
    }

    var allocations = totalAmount.allocate(totalInstallments);

    var installments = new ArrayList<TransactionModel>(totalInstallments);
    for (int i = 0; i < totalInstallments; i++) {
      installments.add(
          TransactionModel.createInstallment(
              description,
              allocations.get(i),
              firstDueDate.plusMonths(i),
              bundleId,
              i + 1,
              totalInstallments,
              user,
              null,
              category));
    }
    return installments;
  }

  public UUID getId() {
    return id;
  }

  public String getDescription() {
    return description;
  }

  public Money getTotalAmount() {
    return totalAmount;
  }

  public Integer getTotalInstallments() {
    return totalInstallments;
  }

  public LocalDate getFirstDueDate() {
    return firstDueDate;
  }

  public com.marcos.meudinheiro.user.domain.model.UserModel getUser() {
    return user;
  }
}
