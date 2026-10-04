package bot

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/bot/fsm"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/client"
)

// handleIncomeCommand processa comandos de entrada de renda/receita (/renda ou /receita).
func (b *Bot) handleIncomeCommand(ctx context.Context, chatID int64, user *tgbotapi.User, text string) {
	cmdPrefix := "/renda"
	if strings.HasPrefix(text, "/receita") {
		cmdPrefix = "/receita"
	}

	if text == cmdPrefix {
		b.handleIncomeHelp(chatID)
		return
	}

	args := strings.TrimSpace(strings.TrimPrefix(text, cmdPrefix))
	amount, desc, err := fsm.ParseExpenseCommand(args)
	if err != nil {
		b.reply(chatID, fmt.Sprintf("⚠️ %s\n\nExemplo correto: `%s 5000 Salário mensal`", err.Error(), cmdPrefix))
		return
	}

	resp, err := b.client.RegisterIncome(ctx, client.IncomeRequest{
		TelegramID:   user.ID,
		Amount:       amount,
		Description:  desc,
		CategoryName: "Renda",
	})
	if err != nil {
		if strings.Contains(err.Error(), "status 404") || strings.Contains(err.Error(), "não encontrado") {
			b.reply(chatID, "⚠️ Você ainda não possui cadastro no *Meu Dinheiro*. Envie /start para configurar seu perfil.")
			return
		}
		b.logger.Error("erro ao registrar receita", "error", err)
		b.reply(chatID, "❌ Erro ao registrar receita na API. Tente novamente mais tarde.")
		return
	}

	diff := resp.NewS2S - resp.PreviousS2S
	healthBadge := formatHealthBadge(resp.HealthStatus)

	msgText := fmt.Sprintf(
		"🎉 *Entrada de R$ %s registrada com sucesso!*\n"+
			"📝 Descrição: *%s*\n\n"+
			"📈 *S2S Anterior:* R$ %s/dia\n"+
			"💰 *Novo S2S:* R$ %s/dia (*+R$ %s/dia*)\n"+
			"💳 *Saldo Líquido Disponível:* R$ %s\n"+
			"📊 *Status do Ciclo:* %s",
		formatMoneyBR(amount),
		desc,
		formatMoneyBR(resp.PreviousS2S),
		formatMoneyBR(resp.NewS2S),
		formatMoneyBR(diff),
		formatMoneyBR(resp.TotalLiquidBalance),
		healthBadge,
	)

	b.replyWithMarkup(chatID, msgText, PersistentMenuKeyboard())
}

func (b *Bot) handleIncomeHelp(chatID int64) {
	msg := "💵 *Registro de Entrada de Dinheiro (Renda/Receita)*\n\n" +
		"Registre salários, recebimentos ou freelas para aumentar seu S2S na hora!\n" +
		"Sintaxe: `/renda <valor> <descrição>` (ou `/receita`)\n\n" +
		"💡 *Exemplos práticos:*\n" +
		"• `/renda 5000 Salário mensal`\n" +
		"• `/renda 450 Pix freela design`\n" +
		"• `/renda 1200 Adiantamento quinzenal`"
	b.replyWithMarkup(chatID, msg, PersistentMenuKeyboard())
}
