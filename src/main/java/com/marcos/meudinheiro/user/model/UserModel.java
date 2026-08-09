package com.marcos.meudinheiro.user.model;


import com.marcos.meudinheiro.shared.valueobjects.Email;
import jakarta.persistence.*;

import java.util.UUID;

@Entity
@Table(name = "tb_users")
public class UserModel {
    @Id
    private UUID id;

    @Column(name = "username", nullable = false, length = 255)
    private String username;

    @Embedded
    private Email email;

    @Column(name = "password", nullable = false, length = 255)
    private  String password;

    public UserModel() {}

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
