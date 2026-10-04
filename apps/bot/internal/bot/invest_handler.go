package bot

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/bot/fsm"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/client"
)

// handlePortfolioCommand exibe a carteira de investimentos consolidada do investidor (D-01..D-04, D-12, D-14, D-15).
func (b *Bot) handlePortfolioCommand(ctx context.Context, chatID int64, userID int64, editMessageID ...int) {
	resp, err := b.client.ListInvestments(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "status 404") || strings.Contains(err.Error(), "não encontrado") {
			b.reply(chatID, "⚠️ Você ainda não possui cadastro no *Meu Dinheiro*. Envie /start para configurar seu perfil.")
			return
		}
		b.logger.Error("erro ao carregar carteira de investimentos", "user_id", userID, "error", err)
		b.reply(chatID, "❌ Não foi possível carregar sua carteira de investimentos no momento. Tente novamente em instantes.")
		return
	}

	// Carteira Vazia (D-04)
	if len(resp.Investments) == 0 {
		emptyMsg := "💼 *Sua Carteira de Ativos está vazia!*\n\n" +
			"Você ainda não possui ações ou títulos cadastrados.\n\n" +
			"Comece agora registrando seus ativos:\n" +
			"• `/investimento ALUP11 100 42.23`\n" +
			"• `/aporte PETR4 50 a 38.50`\n" +
			"• `/comprar TD-SELIC 1 a 14500.00`"

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("➕ Adicionar Ativo", "invest:aporte"),
			),
		)

		if len(editMessageID) > 0 && editMessageID[0] != 0 {
			edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, editMessageID[0], emptyMsg, keyboard)
			edit.ParseMode = tgbotapi.ModeMarkdown
			if _, err := b.send(edit); err != nil {
				b.logger.Warn("falha ao editar mensagem de carteira vazia", "error", err)
			}
			return
		}

		b.replyWithMarkup(chatID, emptyMsg, keyboard)
		return
	}

	// Ordena ativos por volume financeiro total investido decrescente (D-02)
	items := make([]client.InvestmentItem, len(resp.Investments))
	copy(items, resp.Investments)
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].TotalCost > items[j].TotalCost
	})

	var sb strings.Builder
	sb.WriteString("📊 *Sua Carteira de Investimentos*\n\n")

	for _, item := range items {
		alloc := 0.0
		if resp.TotalInvested > 0 {
			alloc = (item.TotalCost / resp.TotalInvested) * 100
		}
		qtyStr := formatQty(float64(item.Quantity))
		sb.WriteString(fmt.Sprintf("🔹 *%s* (%.1f%% da carteira)\n", item.Ticker, alloc))
		sb.WriteString(fmt.Sprintf("• %s cotas • PM: R$ %s • Total: R$ %s\n\n",
			qtyStr,
			formatMoneyBR(item.AveragePrice),
			formatMoneyBR(item.TotalCost),
		))
	}

	sb.WriteString("───────────────────────────\n")
	sb.WriteString(fmt.Sprintf("💼 *Total Investido:* R$ %s\n", formatMoneyBR(resp.TotalInvested)))
	sb.WriteString(fmt.Sprintf("🏦 *Saldo em Conta:* R$ %s\n", formatMoneyBR(resp.TotalLiquidBalance)))
	sb.WriteString(fmt.Sprintf("🌐 *Patrimônio Líquido Total:* R$ %s", formatMoneyBR(resp.TotalNetWorth)))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Novo Aporte", "invest:aporte"),
			tgbotapi.NewInlineKeyboardButtonData("🔄 Atualizar", "invest:refresh"),
		),
	)

	// Edição in-place em refresh para não poluir o histórico do chat (D-14)
	if len(editMessageID) > 0 && editMessageID[0] != 0 {
		edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, editMessageID[0], sb.String(), keyboard)
		edit.ParseMode = tgbotapi.ModeMarkdown
		if _, err := b.send(edit); err != nil {
			b.logger.Warn("falha ao editar mensagem de carteira", "error", err)
		}
		return
	}

	b.replyWithMarkup(chatID, sb.String(), keyboard)
}

// handleInvestCommand processa compras/aportes em ativos (/investimento, /comprar, /aporte) (INVEST-01, D-15, D-16).
func (b *Bot) handleInvestCommand(ctx context.Context, chatID int64, user *tgbotapi.User, text string) {
	cmdPrefix := "/investimento"
	switch {
	case strings.HasPrefix(text, "/comprar"):
		cmdPrefix = "/comprar"
	case strings.HasPrefix(text, "/aporte"):
		cmdPrefix = "/aporte"
	}

	if text == cmdPrefix {
		b.handleInvestHelp(chatID)
		return
	}

	args := strings.TrimSpace(strings.TrimPrefix(text, cmdPrefix))
	invData, err := fsm.ParseInvestment(args)
	if err != nil {
		b.reply(chatID, fmt.Sprintf("⚠️ %s\n\nExemplo correto: `%s ALUP11 100 42.23`", err.Error(), cmdPrefix))
		return
	}

	resp, err := b.client.AddInvestment(ctx, client.AddInvestmentRequest{
		TelegramID: user.ID,
		Ticker:     invData.Ticker,
		Quantity:   invData.Quantity,
		Price:      invData.AveragePrice,
	})
	if err != nil {
		if strings.Contains(err.Error(), "status 404") || strings.Contains(err.Error(), "não encontrado") {
			b.reply(chatID, "⚠️ Você ainda não possui cadastro no *Meu Dinheiro*. Envie /start para configurar seu perfil.")
			return
		}
		b.logger.Error("erro ao registrar aporte", "error", err)
		b.reply(chatID, "❌ Erro ao registrar aporte na API. Tente novamente mais tarde.")
		return
	}

	qtyStr := formatQty(float64(resp.Investment.Quantity))
	msgText := fmt.Sprintf(
		"✅ *Aporte registrado com sucesso!*\n\n"+
			"🏷️ *Ativo:* `%s`\n"+
			"📦 *Custódia Total:* %s cotas\n"+
			"🎯 *Preço Médio Ponderado:* R$ %s\n"+
			"💰 *Posição no Ativo:* R$ %s\n\n"+
			"💼 *Total na Carteira:* R$ %s\n"+
			"🌐 *Patrimônio Líquido Total:* R$ %s",
		resp.Investment.Ticker,
		qtyStr,
		formatMoneyBR(resp.Investment.AveragePrice),
		formatMoneyBR(resp.Investment.TotalCost),
		formatMoneyBR(resp.TotalInvested),
		formatMoneyBR(resp.TotalNetWorth),
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💼 Ver Carteira", "invest:refresh"),
		),
	)

	b.replyWithMarkup(chatID, msgText, keyboard)
}

func (b *Bot) handleInvestHelp(chatID int64) {
	msg := "📈 *Registro de Aporte de Investimento*\n\n" +
		"Atualize sua carteira e recalcule automaticamente o seu Preço Médio Ponderado.\n\n" +
		"*Sintaxe:*\n" +
		"`/investimento <TICKER> <QUANTIDADE> [a] <PREÇO>` (ou `/aporte`, `/comprar`)\n\n" +
		"💡 *Exemplos práticos:*\n" +
		"• `/investimento ALUP11 100 42.23`\n" +
		"• `/aporte PETR4 50 a 38.50`\n" +
		"• `/comprar TD-SELIC 1 a 14500.00`"
	b.replyWithMarkup(chatID, msg, PersistentMenuKeyboard())
}

// handleSellCommand processa baixas e vendas de ativos (/venda, /vender) (INVEST-03, D-05..D-10, D-15, D-16).
func (b *Bot) handleSellCommand(ctx context.Context, chatID int64, user *tgbotapi.User, text string) {
	cmdPrefix := "/venda"
	if strings.HasPrefix(text, "/vender") {
		cmdPrefix = "/vender"
	}

	if text == cmdPrefix {
		b.handleSellHelp(chatID)
		return
	}

	args := strings.TrimSpace(strings.TrimPrefix(text, cmdPrefix))
	ticker, qty, price, err := fsm.ParseSaleCommand(args)
	if err != nil {
		b.reply(chatID, fmt.Sprintf("⚠️ %s\n\nExemplo correto: `%s PETR4 30 a 41.50`", err.Error(), cmdPrefix))
		return
	}

	b.executeSell(ctx, chatID, user.ID, ticker, qty, price)
}

func (b *Bot) executeSell(ctx context.Context, chatID int64, userID int64, ticker string, qty float64, price *float64) {
	resp, err := b.client.SellInvestment(ctx, client.SellInvestmentRequest{
		TelegramID: userID,
		Ticker:     ticker,
		Quantity:   qty,
	})
	if err != nil {
		// Tratamento de excesso de custódia com atalho inteligente (D-07)
		if strings.Contains(err.Error(), "quantidade insuficiente para venda") {
			listResp, listErr := b.client.ListInvestments(ctx, userID)
			if listErr == nil {
				for _, item := range listResp.Investments {
					if strings.EqualFold(item.Ticker, ticker) {
						currentQty := float64(item.Quantity)
						priceArg := "0"
						if price != nil {
							priceArg = fmt.Sprintf("%.2f", *price)
						}
						sellAllData := fmt.Sprintf("invest:sell_all:%s:%s:%s", item.Ticker, formatQty(currentQty), priceArg)

						msg := fmt.Sprintf(
							"⚠️ *Custódia insuficiente para venda de %s!*\n\n"+
								"Você solicitou a venda de %s cotas, mas possui apenas *%s cotas* em custódia.",
							item.Ticker,
							formatQty(qty),
							formatQty(currentQty),
						)

						keyboard := tgbotapi.NewInlineKeyboardMarkup(
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("Vender Todas as %s", formatQty(currentQty)), sellAllData),
								tgbotapi.NewInlineKeyboardButtonData("❌ Cancelar", "invest:cancel"),
							),
						)
						b.replyWithMarkup(chatID, msg, keyboard)
						return
					}
				}
			}
			b.reply(chatID, fmt.Sprintf("⚠️ Você não possui o ativo *%s* em carteira.", ticker))
			return
		}

		if strings.Contains(err.Error(), "status 404") || strings.Contains(err.Error(), "não encontrado") {
			b.reply(chatID, fmt.Sprintf("⚠️ Você não possui o ativo *%s* em carteira.", ticker))
			return
		}

		b.logger.Error("erro ao registrar venda de ativo", "error", err)
		b.reply(chatID, "❌ Erro ao registrar venda na API. Tente novamente mais tarde.")
		return
	}

	soldQty := float64(resp.SoldQuantity)
	remQty := float64(resp.RemainingQuantity)
	pm := resp.AveragePrice

	// Venda COM apuração de preço e P&L (D-05, D-06, D-08, D-09, D-10)
	if price != nil {
		sellPrice := *price
		totalProceeds := sellPrice * soldQty
		pnlAmount := (sellPrice - pm) * soldQty
		pnlPercent := 0.0
		if pm > 0 {
			pnlPercent = ((sellPrice - pm) / pm) * 100
		}

		var pnlLine string
		if pnlAmount >= 0 {
			pnlLine = fmt.Sprintf("🟢 *Lucro Realizado:* +R$ %s (+%.2f%%)", formatMoneyBR(pnlAmount), pnlPercent)
		} else {
			pnlLine = fmt.Sprintf("🔴 *Prejuízo Realizado:* -R$ %s (%.2f%%)", formatMoneyBR(math.Abs(pnlAmount)), pnlPercent)
		}

		var title string
		if resp.IsClosed {
			title = fmt.Sprintf("🏁 *Posição Encerrada!*\n\nTodas as cotas de *%s* foram vendidas.", resp.Ticker)
		} else {
			title = "✅ *Venda registrada com sucesso!*"
		}

		var sb strings.Builder
		sb.WriteString(title + "\n\n")
		sb.WriteString(fmt.Sprintf("🏷️ *Ativo:* `%s`\n", resp.Ticker))
		sb.WriteString(fmt.Sprintf("📦 *Quantidade Vendida:* %s cotas\n", formatQty(soldQty)))
		sb.WriteString(fmt.Sprintf("💵 *Preço de Venda:* R$ %s\n", formatMoneyBR(sellPrice)))
		sb.WriteString(fmt.Sprintf("🎯 *Preço Médio (PM):* R$ %s\n", formatMoneyBR(pm)))
		sb.WriteString(fmt.Sprintf("💰 *Total Apurado:* R$ %s\n", formatMoneyBR(totalProceeds)))
		sb.WriteString(pnlLine + "\n")
		if !resp.IsClosed {
			sb.WriteString(fmt.Sprintf("📦 *Custódia Remanescente:* %s cotas\n", formatQty(remQty)))
		}
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("💼 *Total na Carteira:* R$ %s\n", formatMoneyBR(resp.TotalInvested)))
		sb.WriteString(fmt.Sprintf("🌐 *Patrimônio Líquido Total:* R$ %s\n\n", formatMoneyBR(resp.TotalNetWorth)))
		sb.WriteString("Deseja creditar o valor da venda no seu saldo disponível em conta?")

		creditData := fmt.Sprintf("invest:credit:%.2f:%s", totalProceeds, resp.Ticker)
		keepData := fmt.Sprintf("invest:keep:%s", resp.Ticker)
		undoData := fmt.Sprintf("invest:undo:%s:%s:%.2f", resp.Ticker, formatQty(soldQty), pm)

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("💳 Creditar R$ %s no Saldo Líquido", formatMoneyBR(totalProceeds)), creditData),
				tgbotapi.NewInlineKeyboardButtonData("🛡️ Manter Apenas na Carteira", keepData),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("↩️ Desfazer Venda", undoData),
			),
		)

		b.replyWithMarkup(chatID, sb.String(), keyboard)
		return
	}

	// Venda SEM preço informado (baixa simples de estoque) (D-05, D-10)
	var title string
	if resp.IsClosed {
		title = fmt.Sprintf("🏁 *Posição Encerrada!*\n\nTodas as cotas de *%s* foram baixadas da carteira.", resp.Ticker)
	} else {
		title = "✅ *Baixa de custódia registrada!*"
	}

	var sb strings.Builder
	sb.WriteString(title + "\n\n")
	sb.WriteString(fmt.Sprintf("🏷️ *Ativo:* `%s`\n", resp.Ticker))
	sb.WriteString(fmt.Sprintf("📦 *Quantidade Baixada:* %s cotas\n", formatQty(soldQty)))
	if !resp.IsClosed {
		sb.WriteString(fmt.Sprintf("📦 *Custódia Remanescente:* %s cotas\n", formatQty(remQty)))
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("💼 *Total na Carteira:* R$ %s\n", formatMoneyBR(resp.TotalInvested)))
	sb.WriteString(fmt.Sprintf("🌐 *Patrimônio Líquido Total:* R$ %s\n\n", formatMoneyBR(resp.TotalNetWorth)))
	sb.WriteString(fmt.Sprintf("💡 _Dica: Para apurar lucro ou prejuízo da operação, envie o preço de venda: `/venda %s %s a 42.00`_", resp.Ticker, formatQty(soldQty)))

	undoData := fmt.Sprintf("invest:undo:%s:%s:%.2f", resp.Ticker, formatQty(soldQty), pm)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("↩️ Desfazer Venda", undoData),
		),
	)

	b.replyWithMarkup(chatID, sb.String(), keyboard)
}

func (b *Bot) handleSellHelp(chatID int64) {
	msg := "📉 *Registro de Venda de Ativos*\n\n" +
		"Dê baixa de custódia na sua carteira e apure seu Lucro ou Prejuízo com opção de creditar o valor no seu saldo bancário diário.\n\n" +
		"*Sintaxe:*\n" +
		"`/venda <TICKER> <QUANTIDADE> [a <PREÇO_DE_VENDA>]` (ou `/vender`)\n\n" +
		"💡 *Exemplos práticos:*\n" +
		"• `/venda PETR4 30 a 41.50` _(calcula lucro/prejuízo)_\n" +
		"• `/venda ALUP11 50` _(baixa simples de custódia)_\n" +
		"• `/vender TD-SELIC 1 a 14500.00`"
	b.replyWithMarkup(chatID, msg, PersistentMenuKeyboard())
}

// handleInvestCallback processa os cliques dos botões inline de investimentos (D-06, D-07, D-10, D-14).
func (b *Bot) handleInvestCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	chatID := cb.Message.Chat.ID
	messageID := cb.Message.MessageID
	userID := cb.From.ID
	data := cb.Data

	switch {
	case data == "invest:aporte":
		b.handleInvestHelp(chatID)

	case data == "invest:refresh":
		b.handlePortfolioCommand(ctx, chatID, userID, messageID)

	case strings.HasPrefix(data, "invest:credit:"):
		parts := strings.Split(strings.TrimPrefix(data, "invest:credit:"), ":")
		if len(parts) != 2 {
			b.reply(chatID, "⚠️ Dados de crédito inválidos.")
			return
		}
		amount, err := strconv.ParseFloat(parts[0], 64)
		if err != nil || amount <= 0 {
			b.reply(chatID, "⚠️ Valor de crédito inválido.")
			return
		}
		ticker := parts[1]

		_, err = b.client.RegisterIncome(ctx, client.IncomeRequest{
			TelegramID:   userID,
			Amount:       amount,
			Description:  "Venda de " + ticker,
			CategoryName: "Investimentos",
		})
		if err != nil {
			b.logger.Error("erro ao creditar saldo de venda", "error", err)
			b.reply(chatID, "❌ Erro ao creditar valor no saldo em conta. Tente novamente mais tarde.")
			return
		}

		edit := tgbotapi.NewEditMessageText(chatID, messageID, fmt.Sprintf(
			"✅ *Crédito confirmado!*\n\n"+
				"R$ %s foi creditado no seu saldo bancário disponível (origem: Venda de %s).\n"+
				"Seu Saldo Seguro Diário (S2S) foi recalculado para cima.",
			formatMoneyBR(amount),
			ticker,
		))
		edit.ParseMode = tgbotapi.ModeMarkdown
		if _, err := b.send(edit); err != nil {
			b.logger.Warn("falha ao editar mensagem de crédito", "error", err)
		}

	case strings.HasPrefix(data, "invest:keep:"):
		ticker := strings.TrimPrefix(data, "invest:keep:")
		edit := tgbotapi.NewEditMessageText(chatID, messageID, fmt.Sprintf(
			"🛡️ *Saldo mantido na corretora!*\n\n"+
				"O capital da venda de %s não foi creditado no saldo de conta corrente e não altera o cálculo do seu S2S diário.",
			ticker,
		))
		edit.ParseMode = tgbotapi.ModeMarkdown
		if _, err := b.send(edit); err != nil {
			b.logger.Warn("falha ao editar mensagem de retenção", "error", err)
		}

	case strings.HasPrefix(data, "invest:undo:"):
		parts := strings.Split(strings.TrimPrefix(data, "invest:undo:"), ":")
		if len(parts) != 3 {
			b.reply(chatID, "⚠️ Dados de reversão inválidos.")
			return
		}
		ticker := parts[0]
		qty, err := strconv.ParseFloat(parts[1], 64)
		if err != nil || qty <= 0 {
			b.reply(chatID, "⚠️ Quantidade para reversão inválida.")
			return
		}
		pm, err := strconv.ParseFloat(parts[2], 64)
		if err != nil || pm < 0 {
			b.reply(chatID, "⚠️ Preço para reversão inválido.")
			return
		}

		_, err = b.client.AddInvestment(ctx, client.AddInvestmentRequest{
			TelegramID: userID,
			Ticker:     ticker,
			Quantity:   qty,
			Price:      pm,
		})
		if err != nil {
			b.logger.Error("erro ao desfazer venda", "error", err)
			b.reply(chatID, "❌ Erro ao desfazer venda na API. Tente novamente mais tarde.")
			return
		}

		edit := tgbotapi.NewEditMessageText(chatID, messageID, fmt.Sprintf(
			"↩️ *Venda desfeita com sucesso!*\n\n"+
				"Foram restauradas %s cotas de *%s* ao Preço Médio original de R$ %s.",
			formatQty(qty),
			ticker,
			formatMoneyBR(pm),
		))
		edit.ParseMode = tgbotapi.ModeMarkdown
		if _, err := b.send(edit); err != nil {
			b.logger.Warn("falha ao editar mensagem de reversão", "error", err)
		}

	case strings.HasPrefix(data, "invest:sell_all:"):
		parts := strings.Split(strings.TrimPrefix(data, "invest:sell_all:"), ":")
		if len(parts) != 3 {
			b.reply(chatID, "⚠️ Dados de venda total inválidos.")
			return
		}
		ticker := parts[0]
		qty, err := strconv.ParseFloat(parts[1], 64)
		if err != nil || qty <= 0 {
			b.reply(chatID, "⚠️ Quantidade inválida.")
			return
		}
		priceVal, _ := strconv.ParseFloat(parts[2], 64)
		var pricePtr *float64
		if priceVal > 0 {
			pricePtr = &priceVal
		}

		b.executeSell(ctx, chatID, userID, ticker, qty, pricePtr)

	case data == "invest:cancel":
		edit := tgbotapi.NewEditMessageText(chatID, messageID, "❌ Operação de venda cancelada. Sua carteira permanece inalterada.")
		edit.ParseMode = tgbotapi.ModeMarkdown
		if _, err := b.send(edit); err != nil {
			b.logger.Warn("falha ao editar mensagem de cancelamento", "error", err)
		}
	}
}

func formatQty(q float64) string {
	if q == math.Floor(q) {
		return fmt.Sprintf("%.0f", q)
	}
	return fmt.Sprintf("%.4g", q)
}
