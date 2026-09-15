package com.marcos.meudinheiro.architecture;

import static com.tngtech.archunit.lang.syntax.ArchRuleDefinition.noClasses;
import static com.tngtech.archunit.library.Architectures.layeredArchitecture;

import com.tngtech.archunit.core.importer.ImportOption;
import com.tngtech.archunit.junit.AnalyzeClasses;
import com.tngtech.archunit.junit.ArchTest;
import com.tngtech.archunit.lang.ArchRule;

@AnalyzeClasses(
    packages = "com.marcos.meudinheiro",
    importOptions = ImportOption.DoNotIncludeTests.class)
class ArchitectureTest {

  /**
   * Regra de camadas: domain é puro (só pode depender de shared); application só conversa com
   * domain e shared.
   */
  @ArchTest
  static final ArchRule Camadas =
      layeredArchitecture()
          .consideringOnlyDependenciesInLayers()
          .layer("domain")
          .definedBy("com.marcos.meudinheiro.(*)..domain..")
          .layer("application")
          .definedBy("com.marcos.meudinheiro.(*)..application..")
          .layer("shared")
          .definedBy("com.marcos.meudinheiro.shared..")
          .whereLayer("domain")
          .mayOnlyAccessLayers("shared", "application")
          .whereLayer("application")
          .mayOnlyAccessLayers("shared", "domain");

  /**
   * Regra entre módulos: comunicação só via application.contract — nenhum módulo acessa
   * infraestrutura de outro módulo (AGENTS.md: "nunca via repository alheio"). Uma regra por
   * módulo, para explicitar o módulo ofensor no log.
   */
  @ArchTest
  static final ArchRule IdentityNaoUsaInfraAlheia =
      nenhumClasseAlheiaAcessa("identity", "identity.infraestructure..");

  @ArchTest
  static final ArchRule UserNaoUsaInfraAlheia =
      nenhumClasseAlheiaAcessa("user", "user.infraesctructure..");

  @ArchTest
  static final ArchRule CategoryNaoUsaInfraAlheia =
      nenhumClasseAlheiaAcessa("category", "category.infraestructure..");

  @ArchTest
  static final ArchRule TransactionNaoUsaInfraAlheia =
      nenhumClasseAlheiaAcessa("transaction", "transaction.infraestructure..");

  @ArchTest
  static final ArchRule BankAccountNaoUsaInfraAlheia =
      nenhumClasseAlheiaAcessa("bankaccount", "bankaccount.infraestructure..");

  /** shared é a base de todos os módulos; nunca depende deles. */
  @ArchTest
  static final ArchRule SharedIndependente =
      noClasses()
          .that()
          .resideInAPackage("com.marcos.meudinheiro.shared..")
          .should()
          .dependOnClassesThat()
          .resideInAnyPackage(
              "com.marcos.meudinheiro.identity..",
              "com.marcos.meudinheiro.user..",
              "com.marcos.meudinheiro.category..",
              "com.marcos.meudinheiro.transaction..",
              "com.marcos.meudinheiro.bankaccount..");

  private static ArchRule nenhumClasseAlheiaAcessa(String modulo, String pacoteInfra) {
    return noClasses()
        .that()
        .resideInAnyPackage(
            "com.marcos.meudinheiro.identity..",
            "com.marcos.meudinheiro.user..",
            "com.marcos.meudinheiro.category..",
            "com.marcos.meudinheiro.transaction..",
            "com.marcos.meudinheiro.bankaccount..")
        .and()
        .resideOutsideOfPackage("com.marcos.meudinheiro." + modulo + "..")
        .should()
        .dependOnClassesThat()
        .resideInAPackage("com.marcos.meudinheiro." + pacoteInfra);
  }
}
