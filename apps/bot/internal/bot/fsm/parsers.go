package fsm

import (
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
)

func ParseMoney(input string) (float64, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return 0, errors.New("Valor monetário inválido. Envie um número positivo, como 3500 ou 3500,00.")
	}

	s = strings.ReplaceAll(s, "R$", "")
	s = strings.ReplaceAll(s, "r$", "")
	s = strings.TrimSpace(s)

	hasComma := strings.Contains(s, ",")
	hasDot := strings.Contains(s, ".")

	if hasComma && hasDot {
		if strings.LastIndex(s, ",") > strings.LastIndex(s, ".") {
			s = strings.ReplaceAll(s, ".", "")
			s = strings.ReplaceAll(s, ",", ".")
		} else {
			// Formato americano com vírgula de milhar: 1,250.50 -> remove vírgula
			s = strings.ReplaceAll(s, ",", "")
		}
	} else if hasComma {
		// Formato com vírgula decimal: 3500,50 -> 3500.50
		s = strings.ReplaceAll(s, ",", ".")
	}

	val, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(val) || math.IsInf(val, 0) || val <= 0 {
		return 0, errors.New("Valor monetário inválido. Envie um número positivo, como 3500 ou 3500,00.")
	}

	val = math.Round(val*100) / 100
	return val, nil
}

func ParseCycleDay(input string) (int, error) {
	s := strings.TrimSpace(input)
	day, err := strconv.Atoi(s)
	if err != nil || day < 1 || day > 28 {
		return 0, errors.New("O dia de início do ciclo deve ser entre 1 e 28 (o teto é 28 para consistência de calendário com o mês de fevereiro).")
	}
	return day, nil
}

// ParseFixedExpense extrai descrição e valor de uma entrada de despesa fixa essencial.
// Exemplos aceitos: "Aluguel 1500", "Luz 250,00", "1200 Aluguel", "Aluguel R$ 1500".
func ParseFixedExpense(input string) (*FixedExpenseData, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return nil, errors.New("Informe a descrição e o valor da despesa (ex: 'Aluguel 1500').")
	}

	// Normaliza separadores
	words := strings.Fields(raw)
	if len(words) < 2 {
		return nil, errors.New("Informe a descrição e o valor da despesa (ex: 'Aluguel 1500').")
	}

	var descParts []string
	var amount float64
	var foundAmount bool

	// Tenta identificar qual token é o valor
	for i, w := range words {
		// Ignora tokens "R$" ou "r$"
		if strings.EqualFold(w, "R$") {
			continue
		}
		cleanToken := strings.TrimPrefix(strings.TrimPrefix(w, "R$"), "r$")
		if val, err := ParseMoney(cleanToken); err == nil && !foundAmount {
			amount = val
			foundAmount = true
		} else {
			descParts = append(descParts, words[i])
		}
	}

	if !foundAmount {
		return nil, errors.New("Não foi possível identificar o valor da despesa. Envie no formato 'Descrição Valor' (ex: 'Aluguel 1500').")
	}

	description := strings.TrimSpace(strings.Trim(strings.Join(descParts, " "), ",;:-"))
	if description == "" {
		return nil, errors.New("A descrição da despesa não pode estar vazia.")
	}

	categoryName := inferCategoryName(description)

	return &FixedExpenseData{
		Description:  description,
		Amount:       amount,
		CategoryName: categoryName,
	}, nil
}

func inferCategoryName(desc string) string {
	lower := strings.ToLower(desc)
	switch {
	case strings.Contains(lower, "transporte") || strings.Contains(lower, "combustível") || strings.Contains(lower, "combustivel") || strings.Contains(lower, "gasolina") || strings.Contains(lower, "ônibus") || strings.Contains(lower, "onibus") || strings.Contains(lower, "uber"):
		return "Transporte"
	case strings.Contains(lower, "aluguel") || strings.Contains(lower, "condomínio") || strings.Contains(lower, "condominio") || strings.Contains(lower, "iptu"):
		return "Moradia"
	case strings.Contains(lower, "luz") || strings.Contains(lower, "água") || strings.Contains(lower, "agua") || strings.Contains(lower, "energia") || strings.Contains(lower, "gás") || strings.Contains(lower, "gas encanado") || strings.Contains(lower, "botijão") || lower == "gas" || strings.Contains(lower, "internet") || strings.Contains(lower, "telefone"):
		return "Utilidades"
	case strings.Contains(lower, "mercado") || strings.Contains(lower, "alimentação") || strings.Contains(lower, "alimentacao") || strings.Contains(lower, "comida") || strings.Contains(lower, "feira"):
		return "Alimentação"
	case strings.Contains(lower, "saúde") || strings.Contains(lower, "saude") || strings.Contains(lower, "farmácia") || strings.Contains(lower, "farmacia") || strings.Contains(lower, "médico") || strings.Contains(lower, "medico") || strings.Contains(lower, "plano"):
		return "Saúde"
	default:
		// Capitaliza a primeira letra
		if len(desc) > 0 {
			return strings.ToUpper(desc[:1]) + desc[1:]
		}
		return desc
	}
}

var (
	tickerRegex   = regexp.MustCompile(`^[A-Z0-9]{4,6}$`)
	letterCommaRe = regexp.MustCompile(`([A-Za-z]),`)
	commaSpaceRe  = regexp.MustCompile(`,\s+`)
)

// ParseInvestment extrai ticker, quantidade e preço de strings flexíveis.
// Exemplos aceitos:
// - "ALUP11 10 42.23"
// - "ALUP11, 10 un a 42.23"
// - "PETR4 100 cotas a 38,50"
func ParseInvestment(input string) (*InvestmentData, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return nil, errors.New("Formato de ativo inválido. Envie no formato: TICKER QUANTIDADE PREÇO (ex: 'ALUP11 10 42.23').")
	}

	// Remove vírgulas de separação sintática sem quebrar vírgulas decimais (ex: "42,23")
	cleaned := letterCommaRe.ReplaceAllString(raw, "$1 ")
	cleaned = commaSpaceRe.ReplaceAllString(cleaned, " ")
	tokens := strings.Fields(cleaned)

	var ticker string
	var nums []float64

	for _, tok := range tokens {
		lower := strings.ToLower(tok)
		if lower == "un" || lower == "unidades" || lower == "cotas" || lower == "cota" || lower == "a" || lower == "de" || lower == "r$" {
			continue
		}

		if val, err := ParseMoney(tok); err == nil {
			nums = append(nums, val)
			continue
		}

		// Se não for número, tenta identificar como ticker
		upper := strings.ToUpper(tok)
		if tickerRegex.MatchString(upper) && ticker == "" {
			ticker = upper
		}
	}

	if ticker == "" || len(nums) < 2 {
		return nil, errors.New("Formato de ativo inválido. Envie no formato: TICKER QUANTIDADE PREÇO (ex: 'ALUP11 10 42.23').")
	}

	qty := nums[0]
	price := nums[1]

	return &InvestmentData{
		Ticker:       ticker,
		Quantity:     qty,
		AveragePrice: price,
	}, nil
}
