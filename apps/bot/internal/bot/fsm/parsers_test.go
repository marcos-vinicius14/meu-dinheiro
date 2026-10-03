package fsm

import (
	"testing"
)

func TestParseMoney(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{"número inteiro simples", "100", 100.00, false},
		{"número decimal com ponto", "100.50", 100.50, false},
		{"número decimal com vírgula", "100,50", 100.50, false},
		{"moeda pt-BR com milhar e vírgula", "R$ 1.250,90", 1250.90, false},
		{"moeda minúscula e espaços", "  r$ 3.500,00  ", 3500.00, false},
		{"número com espaços", "  3000  ", 3000.00, false},
		{"formato americano milhar vírgula ponto", "1,250.50", 1250.50, false},
		{"string vazia", "", 0, true},
		{"espaços apenas", "   ", 0, true},
		{"zero", "0", 0, true},
		{"zero decimal", "0.00", 0, true},
		{"negativo", "-50", 0, true},
		{"texto inválido", "abc", 0, true},
		{"prefixo sem número", "R$", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMoney(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseMoney(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseMoney(%q) = %v, want %v", tt.input, got, tt.want)
			}
			if tt.wantErr && err.Error() != "Valor monetário inválido. Envie um número positivo, como 3500 ou 3500,00." {
				t.Errorf("ParseMoney(%q) error message = %q, unexpected", tt.input, err.Error())
			}
		})
	}
}

func TestParseCycleDay(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{"dia 1 limite inferior", "1", 1, false},
		{"dia 15 meio do mês", "15", 15, false},
		{"dia 28 limite superior", "28", 28, false},
		{"dia com zero à esquerda", "01", 1, false},
		{"dia com espaços", "  20  ", 20, false},
		{"dia 0 inválido", "0", 0, true},
		{"dia 29 acima do teto", "29", 0, true},
		{"dia 30 acima do teto", "30", 0, true},
		{"dia 31 acima do teto", "31", 0, true},
		{"número negativo", "-1", 0, true},
		{"texto alfanumérico", "fevereiro", 0, true},
		{"vazio", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCycleDay(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseCycleDay(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseCycleDay(%q) = %v, want %v", tt.input, got, tt.want)
			}
			if tt.wantErr {
				expectedErr := "O dia de início do ciclo deve ser entre 1 e 28 (o teto é 28 para consistência de calendário com o mês de fevereiro)."
				if err.Error() != expectedErr {
					t.Errorf("ParseCycleDay(%q) error message = %q, want %q", tt.input, err.Error(), expectedErr)
				}
			}
		})
	}
}

func TestParseFixedExpense(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantDesc     string
		wantAmount   float64
		wantCategory string
		wantErr      bool
	}{
		{"aluguel comum", "Aluguel 1500", "Aluguel", 1500.00, "Moradia", false},
		{"luz com centavos e vírgula", "Luz 250,00", "Luz", 250.00, "Utilidades", false},
		{"valor antes da descrição", "1200 Aluguel", "Aluguel", 1200.00, "Moradia", false},
		{"internet com R$", "Internet R$ 120,50", "Internet", 120.50, "Utilidades", false},
		{"mercado essencial", "Mercado 800", "Mercado", 800.00, "Alimentação", false},
		{"saúde e farmácia", "Farmácia 150", "Farmácia", 150.00, "Saúde", false},
		{"transporte combustível", "Gasolina 300", "Gasolina", 300.00, "Transporte", false},
		{"categoria desconhecida capitalizada", "Academia 120", "Academia", 120.00, "Academia", false},
		{"com vírgula sintática após descrição", "Aluguel, 1500", "Aluguel", 1500.00, "Moradia", false},
		{"string vazia", "", "", 0, "", true},
		{"apenas descrição sem valor", "Aluguel", "", 0, "", true},
		{"apenas valor sem descrição", "1500", "", 0, "", true},
		{"valor zero", "Aluguel 0", "", 0, "", true},
		{"valor negativo", "Aluguel -50", "", 0, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFixedExpense(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseFixedExpense(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr {
				if got.Description != tt.wantDesc {
					t.Errorf("Description = %q, want %q", got.Description, tt.wantDesc)
				}
				if got.Amount != tt.wantAmount {
					t.Errorf("Amount = %v, want %v", got.Amount, tt.wantAmount)
				}
				if got.CategoryName != tt.wantCategory {
					t.Errorf("CategoryName = %q, want %q", got.CategoryName, tt.wantCategory)
				}
			}
		})
	}
}

func TestParseInvestment(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTk    string
		wantQty   float64
		wantPrice float64
		wantErr   bool
	}{
		{"ticker quantidade preço simples", "ALUP11 10 42.23", "ALUP11", 10, 42.23, false},
		{"com unidades e pontuação", "ALUP11, 10 un a 42.23", "ALUP11", 10, 42.23, false},
		{"com cotas e vírgula decimal", "PETR4 100 cotas a 38,50", "PETR4", 100, 38.50, false},
		{"com de e R$", "VALE3 50 de R$ 65,00", "VALE3", 50, 65.00, false},
		{"ticker minúsculo convertido", "bbas3 20 un a 28,10", "BBAS3", 20, 28.10, false},
		{"string vazia", "", "", 0, 0, true},
		{"apenas ticker", "ALUP11", "", 0, 0, true},
		{"sem ticker", "10 42.23", "", 0, 0, true},
		{"faltando preço", "ALUP11 10", "", 0, 0, true},
		{"quantidade zero", "ALUP11 0 42.23", "", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInvestment(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseInvestment(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr {
				if got.Ticker != tt.wantTk {
					t.Errorf("Ticker = %q, want %q", got.Ticker, tt.wantTk)
				}
				if got.Quantity != tt.wantQty {
					t.Errorf("Quantity = %v, want %v", got.Quantity, tt.wantQty)
				}
				if got.AveragePrice != tt.wantPrice {
					t.Errorf("AveragePrice = %v, want %v", got.AveragePrice, tt.wantPrice)
				}
			}
		})
	}
}
