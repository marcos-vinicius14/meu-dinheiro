package fsm

import (
	"strings"
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

func TestParseExpenseCommand(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantAmount float64
		wantDesc   string
		wantErr    bool
	}{
		{"valor e descrição simples com ponto", "34.90 Almoço", 34.90, "Almoço", false},
		{"valor inteiro e descrição composta", "120 Mercado semanal", 120.00, "Mercado semanal", false},
		{"com prefixo R$ e vírgula", "R$ 45,50 Farmácia", 45.50, "Farmácia", false},
		{"espaços extras e valor no início", "  50   Uber  ", 50.00, "Uber", false},
		{"descrição antes do valor", "Almoço executivo 35.00", 35.00, "Almoço executivo", false},
		{"apenas valor sem descrição assume padrão", "25.00", 25.00, "Despesa rápida", false},
		{"string vazia deve falhar", "", 0, "", true},
		{"espaços em branco deve falhar", "   ", 0, "", true},
		{"apenas texto sem número deve falhar", "apenas texto sem valor", 0, "", true},
		{"valor negativo deve falhar", "-20 Teste", 0, "", true},
		{"valor zero deve falhar", "0 Almoço", 0, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, desc, err := ParseExpenseCommand(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseExpenseCommand(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr {
				if amount != tt.wantAmount {
					t.Errorf("amount = %v, want %v", amount, tt.wantAmount)
				}
				if desc != tt.wantDesc {
					t.Errorf("desc = %q, want %q", desc, tt.wantDesc)
				}
			}
		})
	}
}

func TestParseSimulationCommand(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		wantAmount       float64
		wantInstallments int
		wantErr          bool
	}{
		{"compra à vista simples", "1500", 1500.00, 1, false},
		{"compra à vista com moeda", "R$ 350,00", 350.00, 1, false},
		{"compra parcelada 12x", "2400 12", 2400.00, 12, false},
		{"compra parcelada com centavos 10x", "3500,00 10", 3500.00, 10, false},
		{"espaços extras e 3 parcelas", "  450.00   3  ", 450.00, 3, false},
		{"string vazia deve falhar", "", 0, 0, true},
		{"espaços apenas deve falhar", "   ", 0, 0, true},
		{"texto inválido deve falhar", "abc", 0, 0, true},
		{"parcelas zero deve falhar", "2400 0", 0, 0, true},
		{"parcelas negativas deve falhar", "2400 -5", 0, 0, true},
		{"valor negativo deve falhar", "-500 2", 0, 0, true},
		{"valor zero deve falhar", "0 12", 0, 0, true},
		{"parcelas acima do limite de 48 deve falhar", "2400 60", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, installments, err := ParseSimulationCommand(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseSimulationCommand(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr {
				if amount != tt.wantAmount {
					t.Errorf("amount = %v, want %v", amount, tt.wantAmount)
				}
				if installments != tt.wantInstallments {
					t.Errorf("installments = %v, want %v", installments, tt.wantInstallments)
				}
			}
		})
	}
}

func TestParseInvestment_FixedIncomeAndEquities(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTk    string
		wantQty   float64
		wantPrice float64
	}{
		{"tesouro selic com hífen", "TD-SELIC 1 14500.00", "TD-SELIC", 1, 14500.00},
		{"cdb inter com unidades e pontuação", "CDB-INTER, 10 un a 1000.00", "CDB-INTER", 10, 1000.00},
		{"tesouro ipca com cotas e vírgula decimal", "TD-IPCA29 2 cotas a 3200,50", "TD-IPCA29", 2, 3200.50},
		{"etf bova11", "BOVA11 50 a 115.20", "BOVA11", 50, 115.20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInvestment(tt.input)
			if err != nil {
				t.Fatalf("ParseInvestment(%q) unexpected error = %v", tt.input, err)
			}
			if got.Ticker != tt.wantTk {
				t.Errorf("Ticker = %q, want %q", got.Ticker, tt.wantTk)
			}
			if got.Quantity != tt.wantQty {
				t.Errorf("Quantity = %v, want %v", got.Quantity, tt.wantQty)
			}
			if got.AveragePrice != tt.wantPrice {
				t.Errorf("AveragePrice = %v, want %v", got.AveragePrice, tt.wantPrice)
			}
		})
	}
}

func TestParseSaleCommand_SuccessCases(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTk    string
		wantQty   float64
		wantPrice *float64
	}{
		{"apenas ticker e quantidade sem preço", "PETR4 30", "PETR4", 30, nil},
		{"ticker quantidade com a e preço decimal", "PETR4 30 a 41.50", "PETR4", 30, floatPtr(41.50)},
		{"ticker quantidade e preço sem preposição", "PETR4 30 41.50", "PETR4", 30, floatPtr(41.50)},
		{"com pontuação unidades e vírgula decimal", "PETR4, 30 un a 41,50", "PETR4", 30, floatPtr(41.50)},
		{"renda fixa com hífen", "TD-SELIC 1 a 14500.00", "TD-SELIC", 1, floatPtr(14500.00)},
		{"ticker minúsculo com de e R$", "bbas3 10 de R$ 28,00", "BBAS3", 10, floatPtr(28.00)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticker, qty, price, err := ParseSaleCommand(tt.input)
			if err != nil {
				t.Fatalf("ParseSaleCommand(%q) unexpected error = %v", tt.input, err)
			}
			if ticker != tt.wantTk {
				t.Errorf("ticker = %q, want %q", ticker, tt.wantTk)
			}
			if qty != tt.wantQty {
				t.Errorf("qty = %v, want %v", qty, tt.wantQty)
			}
			if (price == nil && tt.wantPrice != nil) || (price != nil && tt.wantPrice == nil) {
				t.Fatalf("price = %v, want %v", price, tt.wantPrice)
			}
			if price != nil && tt.wantPrice != nil && *price != *tt.wantPrice {
				t.Errorf("price = %v, want %v", *price, *tt.wantPrice)
			}
		})
	}
}

func TestParseSaleCommand_ValidationErrors(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		errSubstr string
	}{
		{"string vazia", "", "Argumentos obrigatórios ausentes"},
		{"sem quantidade", "PETR4", "Informe a quantidade"},
		{"sem ticker", "10 40.00", "Código do ativo (ticker) não informado ou inválido"},
		{"quantidade zero", "PETR4 0 40.00", "Quantidade deve ser maior que zero"},
		{"quantidade negativa", "PETR4 -10 40.00", "Quantidade deve ser maior que zero"},
		{"preço negativo", "PETR4 10 -40.00", "Preço de venda não pode ser negativo"},
		{"ticker muito longo acima de 12", "TICKERMUITOLONGO123 10", "Código do ativo (ticker) não informado ou inválido"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := ParseSaleCommand(tt.input)
			if err == nil {
				t.Fatalf("ParseSaleCommand(%q) expected error containing %q, got nil", tt.input, tt.errSubstr)
			}
			if !strings.Contains(err.Error(), tt.errSubstr) {
				t.Errorf("ParseSaleCommand(%q) error = %q, want substr %q", tt.input, err.Error(), tt.errSubstr)
			}
		})
	}
}

func floatPtr(f float64) *float64 {
	return &f
}
