package tehnikreport

import (
	"strings"

	"fmt"
	"strconv"

	"gopkg.in/telegram-bot-api.v4"
)

// chatbot тип для хранения всего что нужно в одном месте
type chatbot struct {
	db        *Db
	state     *ChatState
	bot       *tgbotapi.BotAPI
	Keyboards []tgbotapi.InlineKeyboardMarkup
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
		tgbotapi.NewInlineKeyboardButtonData("все введено", "return"),
	),
)

// BotInit для иницализации бота
func BotInit(token, datadase string) (*chatbot, error) {
	d, err := Initialize(datadase)
	if err != nil {
		return new(chatbot), err
	}
	s := new(ChatState)
	s.LoadUsers(d.LoadUsers())
	b, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return new(chatbot), err
	}
	kb := make([]tgbotapi.InlineKeyboardMarkup, 0)
	for i := 0; i < 4; i++ {
		kb = append(kb, *GetKeyboard(i))
	}
	kb = append(kb, *GetMaterialsKeyb())
	return &chatbot{db: d, bot: b, state: s, Keyboards: kb}, nil
}

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
		case "services":
			go ch.Services(u.CallbackQuery)
		case "materials":
			go ch.Materials(u.CallbackQuery)
		case "soft":
			go ch.Soft(u.CallbackQuery)
		case "tv":
			go ch.TV(u.CallbackQuery)
		case "cable":
			go ch.Cable(u.CallbackQuery)
		case "router":
			go ch.Router(u.CallbackQuery)
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
		case "cancel", "c":
			go ch.Cancel(u.Message.Chat.ID)
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
func (ch *chatbot) Tiket(m *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(m.Chat.ID, "")
	for _, t := range ch.db.LoadTikets(ch.state.GetUserID(m.Chat.ID)) {
		msg.Text = t.Client
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("отчет", fmt.Sprintf("report%d", t.ID))))
		ch.bot.Send(msg)
	}
}

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
		if stat, uid := ch.db.Login(m.Contact.PhoneNumber, m.Chat.ID); stat {
			msg.Text = "Аутентификация пройдена успешно"
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			ch.state.AddUser(m.Chat.ID, uid)
		} else {
			msg.Text = "К сожалению не удалось найти активного пользователя с данным номером"
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		}
	}
	ch.bot.Send(msg)
}

// NewReport
func (ch *chatbot) NewReport(cal *tgbotapi.CallbackQuery) {
	if strings.HasPrefix(cal.Data, "report") {
		sid := strings.Split(cal.Data, "report")
		id, err := strconv.Atoi(sid[1])
		if err != nil {
			return
		}
		ch.state.SetAction(cal.Message.Chat.ID, "status")
		ch.state.AddReport(cal.Message.Chat.ID, id)
		ch.bot.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: cal.Message.Chat.ID, MessageID: cal.Message.MessageID})
		msg := tgbotapi.NewMessage(cal.Message.Chat.ID, "Заявка выполнена?")
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("да", "true"),
			tgbotapi.NewInlineKeyboardButtonData("нет", "false"),
		))
		ch.bot.Send(msg)
	}
}

// Status
func (ch *chatbot) Status(cal *tgbotapi.CallbackQuery) {
	msg := tgbotapi.NewMessage(cal.Message.Chat.ID, "")
	switch cal.Data {
	case "true":
		ch.state.SetStatus(cal.Message.Chat.ID, true)
		ch.state.SetAction(cal.Message.Chat.ID, "bso")
		msg.Text = "какой номер БСО?"
	case "false":
		ch.state.SetStatus(cal.Message.Chat.ID, false)
		ch.state.SetAction(cal.Message.Chat.ID, "comment")
		msg.Text = "Ваши комметнарии к заявке"
	default:
		return
	}
	ch.bot.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: cal.Message.Chat.ID, MessageID: cal.Message.MessageID})
	ch.bot.Send(msg)
}

// Services
func (ch *chatbot) Services(cal *tgbotapi.CallbackQuery) {
	if ch.state.GetAction(cal.Message.Chat.ID) != "services" {
		return
	}
	msg := tgbotapi.NewMessage(cal.Message.Chat.ID, "выберите выполненные работы (множественный выбор)")
	ch.bot.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: cal.Message.Chat.ID, MessageID: cal.Message.MessageID})
	switch cal.Data {
	case "0":
		ch.state.SetAction(cal.Message.Chat.ID, "soft")
		msg.ReplyMarkup = ch.Keyboards[0]
	case "1":
		ch.state.SetAction(cal.Message.Chat.ID, "cable")
		msg.ReplyMarkup = ch.Keyboards[1]
	case "2":
		ch.state.SetAction(cal.Message.Chat.ID, "tv")
		msg.ReplyMarkup = ch.Keyboards[2]
	case "3":
		ch.state.SetAction(cal.Message.Chat.ID, "router")
		msg.ReplyMarkup = ch.Keyboards[3]
	case "return":
		if ch.state.IsCable(cal.Message.Chat.ID) {
			ch.state.SetAction(cal.Message.Chat.ID, "materials")
			msg.Text = "какие материалы были использованы"
			msg.ReplyMarkup = ch.Keyboards[4]
		} else {
			ch.state.SetAction(cal.Message.Chat.ID, "comment")
			msg.Text = "ваши комментарии к заявке"
		}
	default:
		//ch.state.SetAction(cal.Message.Chat.ID, "soft")
		msg.ReplyMarkup = ServiceTypeKeyb
	}
	ch.bot.Send(msg)
}

// Materials
func (ch *chatbot) Materials(cal *tgbotapi.CallbackQuery) {}

// Bso
func (ch *chatbot) Bso(m *tgbotapi.Message) {
	bso, err := strconv.ParseInt(m.Text, 10, 32)
	msg := tgbotapi.NewMessage(m.Chat.ID, "")
	if err != nil {
		msg.Text = "Не удалось найти номер БСО, попробуйте еще раз"
		ch.bot.Send(msg)
		return
	}
	ch.state.SetBso(m.Chat.ID, int(bso))
	ch.state.SetAction(m.Chat.ID, "amount")
	msg.Text = "Сумма оказанных услуг"
	ch.bot.Send(msg)
}

// Amount
func (ch *chatbot) Amount(m *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(m.Chat.ID, "")
	amount, err := strconv.ParseFloat(m.Text, 32)
	if err != nil {
		msg.Text = "не удалось распознать сумму услуг"
		ch.bot.Send(msg)
		return
	}
	ch.state.SetAmount(m.Chat.ID, float32(amount))
	ch.state.SetAction(m.Chat.ID, "services")
	msg.Text = "Какие услуги были оказаны?"
	msg.ReplyMarkup = ServiceTypeKeyb
	ch.bot.Send(msg)

}

// Comment
func (ch *chatbot) Comment(m *tgbotapi.Message) {
	ch.state.SetComment(m.Chat.ID, m.Text)
}

// Cancel отмена заполнение отчета
func (ch *chatbot) Cancel(chatid int64) {
	ch.state.Clear(chatid)
}

//
func (ch *chatbot) Soft(cal *tgbotapi.CallbackQuery) {
	id, err := strconv.ParseInt(cal.Data, 10, 32)
	if err == nil {
		ch.state.AddService(cal.Message.Chat.ID, &Service{Type: 0, Job: int(id)})
	}
	if cal.Data == "remove" {
		ch.state.SetAction(cal.Message.Chat.ID, "services")
		ch.Services(cal)
	}
}

//
func (ch *chatbot) Cable(cal *tgbotapi.CallbackQuery) {
	id, err := strconv.ParseInt(cal.Data, 10, 32)
	if err == nil {
		ch.state.AddService(cal.Message.Chat.ID, &Service{Type: 1, Job: int(id)})
	}
	if cal.Data == "remove" {
		ch.state.SetAction(cal.Message.Chat.ID, "services")
		ch.Services(cal)
	}
}

//
func (ch *chatbot) TV(cal *tgbotapi.CallbackQuery) {
	id, err := strconv.ParseInt(cal.Data, 10, 32)
	if err == nil {
		ch.state.AddService(cal.Message.Chat.ID, &Service{Type: 2, Job: int(id)})
	}
	if cal.Data == "remove" {
		ch.state.SetAction(cal.Message.Chat.ID, "services")
		ch.Services(cal)
	}
}

//
func (ch *chatbot) Router(cal *tgbotapi.CallbackQuery) {
	id, err := strconv.ParseInt(cal.Data, 10, 32)
	if err == nil {
		ch.state.AddService(cal.Message.Chat.ID, &Service{Type: 3, Job: int(id)})
	}
	if cal.Data == "remove" {
		ch.state.SetAction(cal.Message.Chat.ID, "services")
		ch.Services(cal)
	}
}
