package dateinterval

import (
	"errors"
	"time"
)

// NormalizeDate trunca uma data para meia-noite em UTC (sem hora, minuto, segundo).
func NormalizeDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// DateInterval representa um intervalo inclusivo de datas calendário.
type DateInterval struct {
	startDate time.Time
	endDate   time.Time
}

// New cria um DateInterval validado e normalizado.
func New(start, end time.Time) (DateInterval, error) {
	if start.IsZero() || end.IsZero() {
		return DateInterval{}, errors.New("datas do intervalo são obrigatórias")
	}

	normStart := NormalizeDate(start)
	normEnd := NormalizeDate(end)

	if normStart.After(normEnd) {
		return DateInterval{}, errors.New("data inicial não pode ser após a data final")
	}

	return DateInterval{
		startDate: normStart,
		endDate:   normEnd,
	}, nil
}

// MonthOf cria um intervalo cobrindo o mês calendário inteiro da data informada.
func MonthOf(t time.Time) DateInterval {
	firstDay := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)
	interval, _ := New(firstDay, lastDay)
	return interval
}

// StartDate retorna a data inicial.
func (d DateInterval) StartDate() time.Time {
	return d.startDate
}

// EndDate retorna a data final.
func (d DateInterval) EndDate() time.Time {
	return d.endDate
}

// Contains verifica se a data está dentro do intervalo inclusivo [startDate, endDate].
func (d DateInterval) Contains(date time.Time) bool {
	norm := NormalizeDate(date)
	return !norm.Before(d.startDate) && !norm.After(d.endDate)
}

// DaysRemaining calcula os dias restantes do intervalo a partir de 'from' (inclusivo).
// Retorna no mínimo 1 (Math.max(1, days)).
func (d DateInterval) DaysRemaining(from time.Time) int {
	normFrom := NormalizeDate(from)
	// Diferença em dias completos: (endDate - normFrom) + 1
	days := int(d.endDate.Sub(normFrom).Hours()/24) + 1
	if days < 1 {
		return 1
	}
	return days
}
