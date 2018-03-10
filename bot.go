package tehnikreport

import (
	"net/http"
	"net/url"
	"strings"

	"gopkg.in/telegram-bot-api.v4"
)

// chatbot тип для хранения всего что нужно в одном месте
type chatbot struct {
	db    *Db
	state *ChatState
	bot   *tgbotapi.BotAPI
}

// ServiceTypeKeyb кнопки выбора типа выполненных работ
var ServiceTypeKeyb = tgbotapi.NewInlineKeyboardMarkup(
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(ServiceTypes[0], "0"),
		tgbotapi.NewInlineKeyboardButtonData(ServiceTypes[1], "1"),
	),
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(ServiceTypes[2], "2"),
		tgbotapi.NewInlineKeyboardButtonData(ServiceTypes[3], "3"),
	),
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("все введено", "remove"),
	),
)

// Help функция отправляет сообщение-инструкцию как пользоваться ботом
func (ch *chatbot) Help(m *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(m.Chat.ID, `Бот предназначен для сбора отчетов о выполненных работах.
		для отправки отчетов необходимо авторизваться с помощью комманды /login
		после авторизации можно будет смотреть свои незакрытые заявки коммандой /tiket
		под каждой заявкой есть кнопочка, с помощю которой можно отправить отчет`)
	ch.bot.Send(msg)
	video := tgbotapi.NewVideoUpload(m.Chat.ID, "help.avi")
	video.Caption = "пример как пользоваться"
	ch.bot.Send(video)
}

// ParseUpdate
func (ch *chatbot) ParseUpdate(u *tgbotapi.Update) {
	if u.CallbackQuery != nil {
		switch ch.state.GetAction(u.CallbackQuery.Message.Chat.ID) {
		case "status":
			go ch.Status(u.CallbackQuery)
		case "services", "services0", "services1", "services2", "services3":
			go ch.Services(u.CallbackQuery)
		case "materials":
			go ch.Materials(u.CallbackQuery)
		default:
			go ch.NewReport(u.CallbackQuery)
		}
	}
	if u.Message != nil && u.Message.IsCommand() {
		switch u.Message.Command() {
		case "help", "h":
			go ch.Help(u.Message)
		case "tiket", "t":
			go ch.Tiket(u.Message)
		case "login", "l":
			go ch.Login(u.Message)
		}
	}
	if u.Message != nil && !u.Message.IsCommand() {
		switch ch.state.GetAction(u.Message.Chat.ID) {
		case "bso":
			go ch.Bso(u.Message)
		case "amount":
			go ch.Amount(u.Message)
		case "comment":
			go ch.Comment(u.Message)
		case "login":
			go ch.Login(u.Message)
		}
	}
}

// Tiket
func (ch *chatbot) Tiket(m *tgbotapi.Message) {}

// Login
func (ch *chatbot) Login(m *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(m.Chat.ID, "")
	if m.IsCommand() {
		msg.Text = "отправьте свой номер для авторизации"
		msg.ReplyMarkup = tgbotapi.NewKeyboardButtonContact("мой номер")
		ch.state.SetAction(m.Chat.ID, "login")
	}
	if m.Contact != nil {
		ln := strings.Count(m.Contact.PhoneNumber, "")
		if ln < 11 {
			msg.Text = "Номер телефона слишком короткий"
			return
		}
		phone := "7" + m.Contact.PhoneNumber[ln-11:]
		http.PostForm("https://oauth.telegram.org/auth/request?bot_id=466266277&origin=http%3A%2F%2Fsuz.iqvision.pro&request_access=write", url.Values{"phone": {phone}})
		msg.Text = "Спасибо! Вам будет отправлено сообщение для подверждения, Пожалуйста нажмите \"ACCEPT\""
		msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	}
	ch.bot.Send(msg)
}

// Oauth
func (ch *chatbot) Oauth(u *tgbotapi.Update) {
}

// NewReport
func (ch *chatbot) NewReport(cal *tgbotapi.CallbackQuery) {}

// Status
func (ch *chatbot) Status(cal *tgbotapi.CallbackQuery) {}

// Services
func (ch *chatbot) Services(cal *tgbotapi.CallbackQuery) {}

// Materials
func (ch *chatbot) Materials(cal *tgbotapi.CallbackQuery) {}

// Bso
func (ch *chatbot) Bso(m *tgbotapi.Message) {}

// Amount
func (ch *chatbot) Amount(m *tgbotapi.Message) {}

// Comment
func (ch *chatbot) Comment(m *tgbotapi.Message) {}
