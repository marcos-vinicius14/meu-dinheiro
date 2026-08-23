package com.marcos.meudinheiro.category.domain.model;

import com.marcos.meudinheiro.user.domain.model.UserModel;
import jakarta.persistence.*;
import org.hibernate.annotations.Generated;

import java.util.UUID;

@Entity
@Table(name = "tb_categories")
public class CategoryModel {
    @Id
    @Generated
    @Column(
            name = "id",
            insertable = false,
            updatable = false
    )
    private UUID id;

    @Column(nullable = false, length = 255)
    private String description;

    @Column(nullable = false, length = 255)
    private String icon;

    @ManyToOne(fetch = FetchType.LAZY, optional = false)
    @JoinColumn(name = "user_id", nullable = false)
    private UserModel user;

    public CategoryModel() {}

    public String getDescription() {
        return description;
    }

    public void setDescription(String description) {
        this.description = description;
    }

    public String getIcon() {
        return icon;
    }

    public void setIcon(String icon) {
        this.icon = icon;
    }

    public UserModel getUser() {
        return user;
    }

    public void setUser(UserModel user) {
        this.user = user;
    }
}
