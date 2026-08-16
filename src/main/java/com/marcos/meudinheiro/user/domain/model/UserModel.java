package com.marcos.meudinheiro.user.domain.model;


import com.marcos.meudinheiro.user.domain.valueobject.Email;
import org.hibernate.annotations.Generated;
import jakarta.persistence.*;

import java.util.UUID;

@Entity
@Table(name = "tb_users")
public class UserModel {
    @Id
    @Generated
    @Column(
            name = "id",
            insertable = false,
            updatable = false
    )
    private UUID id;

    @Column(name = "username", nullable = false, length = 255)
    private String username;

    @Embedded
    private Email email;

    @Column(name = "password", nullable = false, length = 255)
    private  String password;

    protected UserModel() {}

    private UserModel(
            String name,
            Email email,
            String password
    ) {
        this.username = name;
        this.email = email;
        this.password = password;
    }

    public static  UserModel create (
            String name,
            Email email,
            String password
    ) {
        return new UserModel(
                name,
                email,
                password
        );
    }

    public String getUsername() {
        return username;
    }

    public void setUsername(String username) {
        this.username = username;
    }

    public Email getEmail() {
        return email;
    }

    public void setEmail(Email email) {
        this.email = email;
    }

    public String getPassword() {
        return password;
    }

    public void setPassword(String password) {
        this.password = password;
    }
}
