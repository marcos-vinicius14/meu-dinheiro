package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/bot/fsm"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/client"
)

// handleSimulateCommand processa comandos de simulação what-if (PRED-03, D-05..D-08, D-15).
func (b *Bot) handleSimulateCommand(ctx context.Context, chatID int64, userID int64, text string) {
	if text == "/simular" || text == ButtonSimulate {
		b.handleSimulateHelp(chatID)
		return
	}

	args := strings.TrimSpace(strings.TrimPrefix(text, "/simular"))
	amount, installments, err := fsm.ParseSimulationCommand(args)
	if err != nil {
		b.reply(chatID, fmt.Sprintf("⚠️ %s\n\nExemplo correto: `/simular 2400 12` ou `/simular 1500`", err.Error()))
		return
	}

	resp, err := b.client.SimulatePurchase(ctx, client.SimulationRequest{
		TelegramID:   userID,
		Amount:       amount,
		Installments: installments,
	})
	if err != nil {
		if strings.Contains(err.Error(), "status 404") || strings.Contains(err.Error(), "não encontrado") {
			b.reply(chatID, "⚠️ Você ainda não possui cadastro no *Meu Dinheiro*. Envie /start para configurar seu perfil.")
			return
		}
		b.logger.Error("erro ao processar simulacao", "error", err)
		b.reply(chatID, "❌ Erro ao simular compra na API. Tente novamente mais tarde.")
		return
	}

	installmentAmount := amount / float64(installments)
	s2sReduction := resp.CurrentCycleS2SBefore - resp.CurrentCycleS2SAfter
	criticalHealthBadge := formatHealthBadge(resp.CriticalCycleHealthStatus)

	// Resumo Executivo (D-05, D-06)
	summaryMsg := fmt.Sprintf(
		"🔮 *Simulação de Compra:* R$ %s (%dx de R$ %s)\n\n"+
			"📅 *Impacto no Ciclo Atual:* S2S cai de R$ %s para *R$ %s/dia* (-R$ %s/dia)\n"+
			"⚠️ *Ciclo Mais Crítico:* Ciclo %d (S2S projetado: R$ %s/dia — %s)",
		formatMoneyBR(amount),
		installments,
		formatMoneyBR(installmentAmount),
		formatMoneyBR(resp.CurrentCycleS2SBefore),
		formatMoneyBR(resp.CurrentCycleS2SAfter),
		formatMoneyBR(s2sReduction),
		resp.CriticalCycleNumber,
		formatMoneyBR(resp.CriticalCycleS2S),
		criticalHealthBadge,
	)

	// Alerta Preventivo de Risco de Déficit (D-08)
	if resp.DeficitRiskAlert {
		recommendation := resp.RecommendationMessage
		if recommendation == "" {
			recommendation = "Considere aumentar o número de parcelas ou adiar a compra."
		}
		summaryMsg += fmt.Sprintf(
			"\n\n🚨 *ALERTA DE RISCO DE DÉFICIT:*\n"+
				"O Ciclo %d fechará no vermelho com essa despesa!\n"+
				"💡 *Recomendação:* %s",
			resp.CriticalCycleNumber,
			recommendation,
		)
	}

	// Teclado Inline com Ação de Confirmação Rápida (D-07)
	amountStr := strconv.FormatFloat(amount, 'f', 2, 64)
	confirmData := fmt.Sprintf("sim_confirm:%s:%d", amountStr, installments)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Confirmar e Lançar", confirmData),
			tgbotapi.NewInlineKeyboardButtonData("❌ Cancelar", "sim_cancel"),
		),
	)

	b.replyWithMarkup(chatID, summaryMsg, keyboard)
}

func (b *Bot) handleSimulateHelp(chatID int64) {
	msg := "🔮 *Simulador What-If de Compras*\n\n" +
		"Simule o impacto de compras futuras em até 12 ciclos antes de passar o cartão!\n" +
		"Sintaxe: `/simular <valor> [parcelas]`\n\n" +
		"💡 *Exemplos:*\n" +
		"• `/simular 1500` (à vista)\n" +
		"• `/simular 2400 12` (12x de R$ 200,00)\n" +
		"• `/simular 450.00 3` (3x de R$ 150,00)"
	b.replyWithMarkup(chatID, msg, PersistentMenuKeyboard())
}

func (b *Bot) handleSimulateCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	chatID := cb.Message.Chat.ID
	user := cb.From

	if cb.Data == "sim_cancel" {
		b.replyWithMarkup(chatID, "❌ Simulação cancelada. Nenhuma transação foi lançada.", PersistentMenuKeyboard())
		return
	}

	if strings.HasPrefix(cb.Data, "sim_confirm:") {
		parts := strings.Split(strings.TrimPrefix(cb.Data, "sim_confirm:"), ":")
		if len(parts) != 2 {
			b.reply(chatID, "⚠️ Dados de simulação inválidos.")
			return
		}

		amount, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			b.reply(chatID, "⚠️ Valor de simulação inválido.")
			return
		}

		installments, err := strconv.Atoi(parts[1])
		if err != nil || installments <= 0 {
			installments = 1
		}

		installmentAmount := amount / float64(installments)
		desc := "Compra simulada"
		if installments > 1 {
			desc = fmt.Sprintf("Compra simulada (1/%d)", installments)
		}

		// Registra a transação via QuickExpense
		_, err = b.client.QuickExpense(ctx, client.QuickExpenseRequest{
			TelegramID:   user.ID,
			Amount:       installmentAmount,
			Description:  desc,
			CategoryName: "Outros",
		})
		if err != nil {
			b.logger.Error("falha ao converter simulacao em compra real", "error", err)
			b.replyWithMarkup(chatID, "❌ Erro ao registrar compra no banco. Tente novamente mais tarde.", PersistentMenuKeyboard())
			return
		}

		successMsg := fmt.Sprintf(
			"🎉 *Compra confirmada e lançada com sucesso!*\n\n"+
				"Registrado: *R$ %s* em %dx de *R$ %s*.\n"+
				"Seu S2S foi atualizado para os próximos ciclos.",
			formatMoneyBR(amount),
			installments,
			formatMoneyBR(installmentAmount),
		)

		b.replyWithMarkup(chatID, successMsg, PersistentMenuKeyboard())
		return
	}
}
