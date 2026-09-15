package com.marcos.meudinheiro.transaction.domain.model;

import com.marcos.meudinheiro.bankaccount.domain.model.BankAccountModel;
import com.marcos.meudinheiro.category.domain.model.CategoryModel;
import com.marcos.meudinheiro.shared.valueobjects.Money;
import com.marcos.meudinheiro.transaction.domain.enums.TransactionStatus;
import com.marcos.meudinheiro.transaction.domain.enums.TransactionType;
import com.marcos.meudinheiro.user.domain.model.UserModel;
import jakarta.persistence.*;
import java.time.LocalDate;
import java.util.Objects;
import java.util.UUID;
import org.hibernate.annotations.Generated;
import org.jspecify.annotations.Nullable;

@Entity
@Table(name = "tb_transactions")
public class TransactionModel {
  @Id
  @Generated
  @Column(name = "id", updatable = false, insertable = false)
  private UUID id;

  @Column(name = "description", length = 500)
  private String description;

  @Embedded
  @AttributeOverride(
      name = "value",
      column = @Column(name = "value", nullable = false, precision = 19, scale = 2))
  private Money amount;

  @Column(name = "type", nullable = false)
  @Enumerated(EnumType.STRING)
  private TransactionType type;

  @Column(name = "status", nullable = false)
  @Enumerated(EnumType.STRING)
  private TransactionStatus status;

  @Column(name = "due_date", nullable = false)
  private LocalDate dueDate;

  @Column(name = "payment_date")
  private LocalDate paymentDate;

  @Column(name = "bundle_id")
  private UUID bundleId;

  @Column(name = "installment_number")
  private Integer installmentNumber;

  @Column(name = "total_installments")
  private Integer totalInstallments;

  @ManyToOne(fetch = FetchType.LAZY, optional = false)
  @JoinColumn(name = "user_id", nullable = false)
  private UserModel user;

  @ManyToOne(fetch = FetchType.LAZY)
  @JoinColumn(name = "bank_account_id")
  private BankAccountModel bankAccount;

  @ManyToOne(fetch = FetchType.LAZY, optional = false)
  @JoinColumn(name = "category_id", nullable = false)
  private CategoryModel category;

  /**
   * Transação avulsa (não-parcelada). Vence como PROJECTED; use confirm/cancel para transitar.
   * user/bankAccount/category são opcionais na factory para permitir testes de regra pura; a
   * persistência exige user e category não-nulos.
   */
  public static TransactionModel create(
      String description,
      TransactionType type,
      @Nullable Money amount,
      LocalDate dueDate,
      UserModel user,
      @Nullable BankAccountModel bankAccount,
      CategoryModel category) {
    if (type == TransactionType.INSTALLMENT_EXPENSE) {
      throw new IllegalArgumentException(
          "Transação parcelada deve ser criada através de um bundle");
    }
    if (amount == null || amount.isNegative() || amount.isZero()) {
      throw new IllegalArgumentException("Valor da transação deve ser positivo");
    }
    if (dueDate == null) {
      throw new IllegalArgumentException("Data de vencimento é obrigatória");
    }
    return new TransactionModel(
        description,
        amount,
        type,
        TransactionStatus.PROJECTED,
        dueDate,
        null,
        null,
        null,
        null,
        user,
        bankAccount,
        category);
  }

  /** Parcela de um bundle: nasce COMMITTED, vínculo obrigatório com o bundle. */
  public static TransactionModel createInstallment(
      String description,
      Money amount,
      LocalDate dueDate,
      UUID bundleId,
      int installmentNumber,
      int totalInstallments,
      UserModel user,
      @Nullable BankAccountModel bankAccount,
      CategoryModel category) {
    if (bundleId == null) {
      throw new IllegalArgumentException("Bundle é obrigatório para parcela");
    }
    if (installmentNumber < 1 || installmentNumber > totalInstallments) {
      throw new IllegalArgumentException(
          "Número da parcela deve estar entre 1 e o total de parcelas");
    }
    if (amount == null || amount.isNegative() || amount.isZero()) {
      throw new IllegalArgumentException("Valor da transação deve ser positivo");
    }
    if (dueDate == null) {
      throw new IllegalArgumentException("Data de vencimento é obrigatória");
    }
    return new TransactionModel(
        description,
        amount,
        TransactionType.INSTALLMENT_EXPENSE,
        TransactionStatus.COMMITTED,
        dueDate,
        null,
        bundleId,
        installmentNumber,
        totalInstallments,
        user,
        bankAccount,
        category);
  }

  protected TransactionModel() {}

  private TransactionModel(
      String description,
      Money amount,
      TransactionType type,
      TransactionStatus status,
      LocalDate dueDate,
      @Nullable LocalDate paymentDate,
      @Nullable UUID bundleId,
      @Nullable Integer installmentNumber,
      @Nullable Integer totalInstallments,
      UserModel user,
      @Nullable BankAccountModel bankAccount,
      CategoryModel category) {
    this.description = description;
    this.amount = amount;
    this.type = type;
    this.status = status;
    this.dueDate = dueDate;
    this.paymentDate = paymentDate;
    this.bundleId = bundleId;
    this.installmentNumber = installmentNumber;
    this.totalInstallments = totalInstallments;
    this.user = user;
    this.bankAccount = bankAccount;
    this.category = category;
  }

  /**
   * Confirma a liquidação monetária. Idempotente: reconfirmação é ignorada silenciosamente
   * (independente da data informada). Permite pagamento antecipado (data anterior ao vencimento).
   * Rejeita transação cancelada.
   */
  public void confirm(LocalDate paymentDate) {
    if (status == TransactionStatus.CANCELED) {
      throw new IllegalStateException("Transação cancelada não pode ser confirmada");
    }
    if (status == TransactionStatus.CONFIRMED) {
      return;
    }
    if (paymentDate == null) {
      throw new IllegalArgumentException("Data de pagamento é obrigatória");
    }
    this.status = TransactionStatus.CONFIRMED;
    this.paymentDate = paymentDate;
  }

  /** Aborta a transação sem impacto de fluxo. Rejeita transação já confirmada. */
  public void cancel() {
    if (status == TransactionStatus.CONFIRMED) {
      throw new IllegalStateException("Transação confirmada não pode ser cancelada");
    }
    this.status = TransactionStatus.CANCELED;
  }

  public void updateDescription(String description) {
    this.description = description;
  }

  public void updateAmount(Money amount) {
    if (amount == null || amount.isNegative() || amount.isZero()) {
      throw new IllegalArgumentException("Valor da transação deve ser positivo");
    }
    this.amount = amount;
  }

  public void updateDueDate(LocalDate dueDate) {
    if (dueDate == null) {
      throw new IllegalArgumentException("Data de vencimento é obrigatória");
    }
    this.dueDate = dueDate;
  }

  public void updateType(TransactionType type) {
    if (type == TransactionType.INSTALLMENT_EXPENSE) {
      throw new IllegalArgumentException("Tipo INSTALLMENT_EXPENSE é gerenciado pelo bundle");
    }
    this.type = Objects.requireNonNull(type);
  }

  public void assignBankAccount(@Nullable BankAccountModel bankAccount) {
    this.bankAccount = bankAccount;
  }

  /** Redefine a categoria (uso do use case de update; parcelas de bundle não passam por aqui). */
  public void setCategoryRef(CategoryModel category) {
    this.category = Objects.requireNonNull(category);
  }

  public boolean belongsTo(UUID userId) {
    return user != null && user.getId().equals(userId);
  }

  public UUID getId() {
    return id;
  }

  public String getDescription() {
    return description;
  }

  public Money getAmount() {
    return amount;
  }

  public TransactionType getType() {
    return type;
  }

  public TransactionStatus getStatus() {
    return status;
  }

  public LocalDate getDueDate() {
    return dueDate;
  }

  public LocalDate getPaymentDate() {
    return paymentDate;
  }

  public UUID getBundleId() {
    return bundleId;
  }

  public Integer getInstallmentNumber() {
    return installmentNumber;
  }

  public Integer getTotalInstallments() {
    return totalInstallments;
  }

  public UserModel getUser() {
    return user;
  }

  public BankAccountModel getBankAccount() {
    return bankAccount;
  }

  public CategoryModel getCategory() {
    return category;
  }
}
