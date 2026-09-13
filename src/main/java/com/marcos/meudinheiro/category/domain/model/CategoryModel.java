package com.marcos.meudinheiro.category.domain.model;

import com.marcos.meudinheiro.category.domain.valueobject.CategoryDescription;
import com.marcos.meudinheiro.category.domain.valueobject.CategoryIcon;
import com.marcos.meudinheiro.user.domain.model.UserModel;
import jakarta.persistence.*;
import org.hibernate.annotations.Generated;

import java.util.Objects;
import java.util.UUID;

@Entity
@Table(name = "tb_categories")
public class CategoryModel {
    @Id
    @Generated
    @Column(name = "id", insertable = false, updatable = false)
    private UUID id;

    @Column(nullable = false, length = 255)
    private String description;

    @Column(length = 255)
    private String icon;

    @ManyToOne(fetch = FetchType.LAZY, optional = false)
    @JoinColumn(name = "user_id", nullable = false)
    private UserModel user;

    protected CategoryModel() {
    }

    private CategoryModel(
            String description,
            String icon,
            UserModel user) {
        this.description = description;
        this.icon = icon;
        this.user = user;
    }

    public static CategoryModel create(
            CategoryDescription description,
            CategoryIcon icon,
            UserModel user) {
        return new CategoryModel(
                description.value(),
                icon != null ? icon.value() : null,
                user);
    }

    public void updateDescription(CategoryDescription description) {
        this.description = Objects.requireNonNull(description).value();
    }

    public void updateIcon(CategoryIcon icon) {
        this.icon = icon != null ? icon.value() : null;
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

    public String getIcon() {
        return icon;
    }

    public UserModel getUser() {
        return user;
    }
}
