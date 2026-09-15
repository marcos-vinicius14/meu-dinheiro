package com.marcos.meudinheiro;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.scheduling.annotation.EnableScheduling;

@SpringBootApplication
@EnableScheduling
public class MeuDinheiroApiApplication {

  public static void main(String[] args) {
    SpringApplication.run(MeuDinheiroApiApplication.class, args);
  }
}
