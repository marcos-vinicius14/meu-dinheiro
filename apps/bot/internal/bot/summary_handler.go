package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/bot/fsm"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/client"
)

// handleSummaryCommand processa comandos de resumo financeiro (/resumo, /resumo hoje, /resumo semanal).
func (b *Bot) handleSummaryCommand(ctx context.Context, chatID int64, userID int64, text string) {
	args := strings.TrimSpace(strings.TrimPrefix(text, "/resumo"))
	period, err := fsm.ParseSummaryCommand(args)
	if err != nil {
		b.reply(chatID, fmt.Sprintf("⚠️ %s", err.Error()))
		return
	}

	if period == "week" {
		b.handleWeeklyCommand(ctx, chatID, userID)
		return
	}

	resp, err := b.client.GetSummaryReport(ctx, userID, "today")
	if err != nil {
		if strings.Contains(err.Error(), "status 404") || strings.Contains(err.Error(), "não encontrado") {
			b.reply(chatID, "⚠️ Você ainda não possui cadastro no *Meu Dinheiro*. Envie /start para configurar seu perfil em 2 minutos!")
			return
		}
		b.logger.Error("erro ao consultar resumo diario", "error", err)
		b.reply(chatID, "❌ Erro ao consultar resumo diário. Tente novamente em instantes.")
		return
	}

	msg := formatDailySummaryMessage(resp)
	b.replyWithMarkup(chatID, msg, PersistentMenuKeyboard())
}

// handleWeeklyCommand processa a consulta do resumo semanal (/semanal ou /resumo semanal).
func (b *Bot) handleWeeklyCommand(ctx context.Context, chatID int64, userID int64) {
	resp, err := b.client.GetSummaryReport(ctx, userID, "week")
	if err != nil {
		if strings.Contains(err.Error(), "status 404") || strings.Contains(err.Error(), "não encontrado") {
			b.reply(chatID, "⚠️ Você ainda não possui cadastro no *Meu Dinheiro*. Envie /start para configurar seu perfil em 2 minutos!")
			return
		}
		b.logger.Error("erro ao consultar resumo semanal", "error", err)
		b.reply(chatID, "❌ Erro ao consultar resumo semanal. Tente novamente em instantes.")
		return
	}

	msg := formatWeeklySummaryMessage(resp)
	b.replyWithMarkup(chatID, msg, PersistentMenuKeyboard())
}

func formatDailySummaryMessage(resp *client.SummaryReportResponse) string {
	var sb strings.Builder

	dateStr := formatDateBR(time.Now().Format("2006-01-02"))
	totalSpent := 0.0
	txCount := 0
	var transactions []client.DaySummaryItem

	if resp.Today != nil {
		dateStr = formatDateBR(resp.Today.Date)
		totalSpent = resp.Today.TotalSpent
		txCount = resp.Today.TransactionsCount
		transactions = resp.Today.Transactions
	}

	healthBadge := formatHealthBadge(resp.HealthStatus)

	sb.WriteString(fmt.Sprintf("📅 *Resumo de Hoje (%s)*\n\n", dateStr))
	sb.WriteString(fmt.Sprintf("💳 *Saldo em Conta:* R$ %s\n", formatMoneyBR(resp.AccountBalance)))
	sb.WriteString(fmt.Sprintf("💸 *Total Gasto Hoje:* R$ %s (%s)\n", formatMoneyBR(totalSpent), formatTransactionsCount(txCount)))
	sb.WriteString(fmt.Sprintf("💰 *S2S para Hoje:* R$ %s/dia\n", formatMoneyBR(resp.S2SToday)))
	sb.WriteString(fmt.Sprintf("📊 *Status do Ciclo:* %s (%d dias restantes)\n\n", healthBadge, resp.DaysRemaining))

	sb.WriteString("📋 *Extrato Rápido do Dia:*\n")
	if len(transactions) == 0 {
		sb.WriteString("_Nenhum gasto registrado hoje até o momento._")
	} else {
		for _, tx := range transactions {
			cat := tx.CategoryName
			if cat == "" {
				cat = "Geral"
			}
			sb.WriteString(fmt.Sprintf("• %s (%s) — R$ %s\n", EscapeMarkdown(tx.Description), EscapeMarkdown(cat), formatMoneyBR(tx.Amount)))
		}
	}

	return strings.TrimSpace(sb.String())
}

func formatWeeklySummaryMessage(resp *client.SummaryReportResponse) string {
	var sb strings.Builder

	periodStr := "Semana Corrente"
	totalSpent := 0.0
	dailyAvg := 0.0
	txCount := 0
	daysElapsed := 1
	var topCategories []client.CategorySummaryItem

	if resp.Week != nil {
		startShort := formatShortDateBR(resp.Week.StartDate)
		endShort := formatShortDateBR(resp.Week.EndDate)
		periodStr = fmt.Sprintf("%s a %s", startShort, endShort)
		totalSpent = resp.Week.TotalSpent
		dailyAvg = resp.Week.DailyAverage
		txCount = resp.Week.TransactionsCount
		daysElapsed = resp.Week.DaysElapsed
		topCategories = resp.Week.TopCategories
	}

	healthBadge := formatHealthBadge(resp.HealthStatus)

	sb.WriteString(fmt.Sprintf("📊 *Resumo Semanal (%s)*\n\n", periodStr))
	sb.WriteString(fmt.Sprintf("💳 *Saldo Atual em Conta:* R$ %s\n", formatMoneyBR(resp.AccountBalance)))
	sb.WriteString(fmt.Sprintf("💸 *Total Gasto na Semana:* R$ %s (%s)\n", formatMoneyBR(totalSpent), formatTransactionsCount(txCount)))
	sb.WriteString(fmt.Sprintf("📉 *Média Diária (%s):* R$ %s/dia\n", formatDaysElapsed(daysElapsed), formatMoneyBR(dailyAvg)))
	sb.WriteString(fmt.Sprintf("💰 *S2S Diário Atual:* R$ %s/dia\n", formatMoneyBR(resp.S2SToday)))
	sb.WriteString(fmt.Sprintf("📊 *Status do Ciclo:* %s\n\n", healthBadge))

	sb.WriteString("🏷️ *Top Categorias de Gastos:*\n")
	if len(topCategories) == 0 {
		sb.WriteString("_Nenhum gasto registrado nesta semana._")
	} else {
		for i, cat := range topCategories {
			pctStr := strings.Replace(fmt.Sprintf("%.1f", float64(cat.Percentage)), ".", ",", 1)
			sb.WriteString(fmt.Sprintf("%d. %s: R$ %s (%s%%)\n", i+1, EscapeMarkdown(cat.CategoryName), formatMoneyBR(cat.Amount), pctStr))
		}
	}

	return strings.TrimSpace(sb.String())
}

func formatDateBR(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	return t.Format("02/01/2006")
}

func formatShortDateBR(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	return t.Format("02/01")
}

func formatTransactionsCount(count int) string {
	if count == 1 {
		return "1 lançamento"
	}
	return fmt.Sprintf("%d lançamentos", count)
}

func formatDaysElapsed(days int) string {
	if days == 1 {
		return "1 dia decorrido"
	}
	return fmt.Sprintf("%d dias decorridos", days)
}
