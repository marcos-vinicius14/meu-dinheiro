package com.marcos.meudinheiro.category.domain.model;

import static org.assertj.core.api.Assertions.assertThat;

import java.util.UUID;

import com.marcos.meudinheiro.category.domain.valueobject.CategoryDescription;
import com.marcos.meudinheiro.category.domain.valueobject.CategoryIcon;
import com.marcos.meudinheiro.user.domain.model.UserModel;
import com.marcos.meudinheiro.user.domain.valueobject.Email;
import org.junit.jupiter.api.Test;

class CategoryModelTest {

    @Test
    void createCategoryStoresValues() {
        var user = createUser();
        var category = CategoryModel.create(
                CategoryDescription.create("Alimentação").value(),
                CategoryIcon.create("food").value(),
                user
        );

        assertThat(category.getDescription()).isEqualTo("Alimentação");
        assertThat(category.getIcon()).isEqualTo("food");
        assertThat(category.getUser()).isEqualTo(user);
    }

    @Test
    void createCategoryWithNullIconStoresNullIcon() {
        var category = CategoryModel.create(
                CategoryDescription.create("Transporte").value(),
                null,
                createUser()
        );

        assertThat(category.getIcon()).isNull();
    }

    @Test
    void belongsToReturnsTrueForSameUser() {
        var user = createUser();
        var category = CategoryModel.create(
                CategoryDescription.create("Lazer").value(),
                null,
                user
        );

        assertThat(category.belongsTo(user.getId())).isTrue();
    }

    @Test
    void belongsToReturnsFalseForDifferentUser() {
        var owner = createUser();
        var other = createUser();
        var category = CategoryModel.create(
                CategoryDescription.create("Lazer").value(),
                null,
                owner
        );

        assertThat(category.belongsTo(other.getId())).isFalse();
    }

    @Test
    void updateDescriptionChangesDescription() {
        var category = CategoryModel.create(
                CategoryDescription.create("Antiga").value(),
                null,
                createUser()
        );

        category.updateDescription(CategoryDescription.create("Nova").value());

        assertThat(category.getDescription()).isEqualTo("Nova");
    }

    @Test
    void updateIconChangesIcon() {
        var category = CategoryModel.create(
                CategoryDescription.create("Lazer").value(),
                CategoryIcon.create("old").value(),
                createUser()
        );

        category.updateIcon(CategoryIcon.create("new").value());

        assertThat(category.getIcon()).isEqualTo("new");
    }

    @Test
    void updateIconWithNullRemovesIcon() {
        var category = CategoryModel.create(
                CategoryDescription.create("Lazer").value(),
                CategoryIcon.create("old").value(),
                createUser()
        );

        category.updateIcon(null);

        assertThat(category.getIcon()).isNull();
    }

    private UserModel createUser() {
        var email = Email.create("user-" + UUID.randomUUID() + "@example.com");
        var user = UserModel.create(
                "Test User",
                email.value(),
                "password123"
        );
        setUserId(user, UUID.randomUUID());
        return user;
    }

    private void setUserId(UserModel user, UUID id) {
        try {
            var field = UserModel.class.getDeclaredField("id");
            field.setAccessible(true);
            field.set(user, id);
        } catch (NoSuchFieldException | IllegalAccessException e) {
            throw new RuntimeException(e);
        }
    }
}
