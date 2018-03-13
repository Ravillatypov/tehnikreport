package tehnikreport

import (
	"strings"

	"fmt"
	"log"
	"strconv"

	"gopkg.in/telegram-bot-api.v4"
)

// ChatBot тип для хранения всего что нужно в одном месте
type ChatBot struct {
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
func BotInit(token, datadase string) (*ChatBot, error) {
	log.Println("BotInit")
	d, err := Initialize(datadase)
	if err != nil {
		log.Println(err.Error())
		return new(ChatBot), err
	}
	s := &ChatState{
		reports: make(map[int64]Report),
		super:   make([]int64, 0),
		action:  make(map[int64]string),
		users:   make(map[int64]uint16),
	}
	s.LoadUsers(d.LoadUsers())
	b, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Println(err.Error())
		return new(ChatBot), err
	}
	kb := make([]tgbotapi.InlineKeyboardMarkup, 0)
	for i := 0; i < 4; i++ {
		kb = append(kb, *GetKeyboard(i))
	}
	kb = append(kb, *GetMaterialsKeyb())
	return &ChatBot{db: d, bot: b, state: s, Keyboards: kb}, nil
}

// Help функция отправляет сообщение-инструкцию как пользоваться ботом
func (ch *ChatBot) Help(m *tgbotapi.Message) {
	log.Println("Help", *m)
	msg := tgbotapi.NewMessage(m.Chat.ID, `Бот предназначен для сбора отчетов о выполненных работах.
		для отправки отчетов необходимо авторизваться с помощью комманды /login
		после авторизации можно будет смотреть свои незакрытые заявки коммандой /tiket
		под каждой заявкой есть кнопочка, с помощю которой можно отправить отчет`)
	ch.bot.Send(msg)
	video := tgbotapi.NewVideoUpload(m.Chat.ID, "help.mp4")
	video.Caption = "пример как пользоваться"
	_, err := ch.bot.Send(video)
	if err != nil {
		log.Println(err.Error())
	}
}

// ParseUpdate по состоянию чата пользователя раздает задачи функциям
func (ch *ChatBot) ParseUpdate(u *tgbotapi.Update) {
	log.Println("ParseUpdate", *u)
	if u.CallbackQuery != nil {
		switch ch.state.GetAction(u.CallbackQuery.Message.Chat.ID) {
		case "status":
			go ch.Status(u.CallbackQuery)
		case "services":
			go ch.Services(u.CallbackQuery)
		case "materials":
			go ch.Materials(u)
		case "soft":
			go ch.Soft(u.CallbackQuery)
		case "tv":
			go ch.TV(u.CallbackQuery)
		case "cable":
			go ch.Cable(u.CallbackQuery)
		case "router":
			go ch.Router(u.CallbackQuery)
		case "dopservices":
			go ch.DopServices(u)
		default:
			go ch.NewReport(u.CallbackQuery)
		}
	}
	if u.Message != nil && u.Message.IsCommand() {
		switch u.Message.Command() {
		case "help", "h", "start":
			go ch.Help(u.Message)
		case "tiket", "t":
			go ch.Tiket(u.Message)
		case "login", "l":
			go ch.Login(u.Message)
		case "cancel", "c":
			go ch.Cancel(u.Message.Chat.ID)
		case "super", "s":
			go ch.Super(u.Message)
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
		case "materials":
			go ch.Materials(u)
		case "login":
			go ch.Login(u.Message)
		case "super":
			go ch.Super(u.Message)
		case "dopservices":
			go ch.DopServices(u)
		}
	}
}

// Tiket получает список заявок пользователя и отправляет с кнопкой для отчета
func (ch *ChatBot) Tiket(m *tgbotapi.Message) {
	log.Println("Tiket", *m)
	msg := tgbotapi.NewMessage(m.Chat.ID, "нет открытых заявок")
	uid := ch.state.GetUserID(m.Chat.ID)
	tikets := ch.db.LoadTikets(uid)
	log.Println("Tiket", uid, tikets)
	if len(tikets) > 0 {
		for _, t := range tikets {
			msg.Text = t.Client
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("отчет", fmt.Sprintf("report%d", t.ID))))
			ch.bot.Send(msg)
			return
		}
	}
	ch.bot.Send(msg)
}

// Login авторизация
func (ch *ChatBot) Login(m *tgbotapi.Message) {
	log.Println("Login", *m)
	msg := tgbotapi.NewMessage(m.Chat.ID, "")
	if m.IsCommand() {
		msg.Text = "отправьте свой номер для авторизации"
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButtonContact("мой номер")))
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
			ch.state.SetAction(m.Chat.ID, "")
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			ch.state.AddUser(m.Chat.ID, uid)
		} else {
			msg.Text = "К сожалению не удалось найти активного пользователя с данным номером"
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		}
	}
	ch.bot.Send(msg)
}

// Super авторизация
func (ch *ChatBot) Super(m *tgbotapi.Message) {
	log.Println("Super", *m)
	msg := tgbotapi.NewMessage(m.Chat.ID, "")
	if m.IsCommand() {
		msg.Text = "отправьте свой номер для авторизации"
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButtonContact("мой номер")))
		ch.state.SetAction(m.Chat.ID, "super")
	}
	if m.Contact != nil {
		ln := strings.Count(m.Contact.PhoneNumber, "")
		if ln < 11 {
			msg.Text = "Номер телефона слишком короткий"
			return
		}
		if stat, _ := ch.db.Login(m.Contact.PhoneNumber, m.Chat.ID); stat {
			msg.Text = "Аутентификация пройдена успешно"
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			ch.state.AddSuper(m.Chat.ID)
			ch.state.Clear(m.Chat.ID)
		} else {
			msg.Text = "К сожалению не удалось найти активного пользователя с данным номером"
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			ch.state.Clear(m.Chat.ID)
		}
	}
	ch.bot.Send(msg)
}

// NewReport создает новый пустой отчет
func (ch *ChatBot) NewReport(cal *tgbotapi.CallbackQuery) {
	log.Println("NewReport", *cal)
	if strings.HasPrefix(cal.Data, "report") {
		sid := strings.Split(cal.Data, "report")
		id, err := strconv.ParseUint(sid[1], 10, 32)
		if err != nil {
			return
		}
		ch.state.SetAction(cal.Message.Chat.ID, "status")
		ch.state.AddReport(cal.Message.Chat.ID, uint32(id))
		ch.bot.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: cal.Message.Chat.ID, MessageID: cal.Message.MessageID})
		msg := tgbotapi.NewMessage(cal.Message.Chat.ID, "Заявка выполнена?")
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("да", "true"),
			tgbotapi.NewInlineKeyboardButtonData("нет", "false"),
		))
		ch.bot.Send(msg)
	}
}

// Status получает статус заявки от пльзователя
func (ch *ChatBot) Status(cal *tgbotapi.CallbackQuery) {
	log.Println("Status", *cal)
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

// DopServices доп услуги
func (ch *ChatBot) DopServices(u *tgbotapi.Update) {
	log.Println("DopServices", *u)
	if u.CallbackQuery != nil {
		ch.bot.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: u.CallbackQuery.Message.Chat.ID,
			MessageID: u.CallbackQuery.Message.MessageID})
		msg := tgbotapi.NewMessage(u.CallbackQuery.Message.Chat.ID, "были дополнительные услуги?")
		switch u.CallbackQuery.Data {
		case "true":
			msg.Text = "напишите список оказанных услуг"
		case "false":
			if ch.state.IsCable(u.CallbackQuery.Message.Chat.ID) {
				ch.state.SetAction(u.CallbackQuery.Message.Chat.ID, "materials")
				msg.Text = "какие материалы были использованы"
				msg.ReplyMarkup = ch.Keyboards[4]
			} else {
				ch.state.SetAction(u.CallbackQuery.Message.Chat.ID, "comment")
				msg.Text = "ваши комментарии к заявке"
			}
		default:
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("да", "true"),
					tgbotapi.NewInlineKeyboardButtonData("нет", "false"),
				),
			)
		}
		ch.bot.Send(msg)
	}
	if u.Message != nil {
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, "")
		ch.state.SetDopServices(u.Message.Chat.ID, u.Message.Text)
		if ch.state.IsCable(u.Message.Chat.ID) {
			ch.state.SetAction(u.Message.Chat.ID, "materials")
			msg.Text = "какие материалы были использованы"
			msg.ReplyMarkup = ch.Keyboards[4]
		} else {
			ch.state.SetAction(u.Message.Chat.ID, "comment")
			msg.Text = "ваши комментарии к заявке"
		}
		ch.bot.Send(msg)
	}

}

// Services получает списой выполненных работ
func (ch *ChatBot) Services(cal *tgbotapi.CallbackQuery) {
	log.Println("Services", *cal)
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
		ch.state.SetAction(cal.Message.Chat.ID, "dopservices")
		ch.DopServices(&tgbotapi.Update{CallbackQuery: cal})
		return
	default:
		//ch.state.SetAction(cal.Message.Chat.ID, "soft")
		msg.ReplyMarkup = ServiceTypeKeyb
	}
	ch.bot.Send(msg)
}

// Materials получает список материалов
func (ch *ChatBot) Materials(u *tgbotapi.Update) {
	log.Println("Materials", *u)
	if u.CallbackQuery != nil {
		id, err := strconv.ParseUint(u.CallbackQuery.Data, 10, 32)
		if err == nil {
			ch.state.AddMaterials(u.CallbackQuery.Message.Chat.ID, &Material{ID: uint8(id)})
			ch.bot.Send(tgbotapi.NewMessage(u.CallbackQuery.Message.Chat.ID, "количество?"))
			return
		}
		if u.CallbackQuery.Data == "remove" {
			ch.bot.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: u.CallbackQuery.Message.Chat.ID, MessageID: u.CallbackQuery.Message.MessageID})
			ch.state.SetAction(u.CallbackQuery.Message.Chat.ID, "comment")
			msg := tgbotapi.NewMessage(u.CallbackQuery.Message.Chat.ID, "Ваши комментари к заявке")
			ch.bot.Send(msg)
		}
	}
	if u.Message != nil {
		count, err := strconv.ParseUint(u.Message.Text, 10, 32)
		if err == nil {
			ch.state.SetMaterialsCount(u.Message.Chat.ID, uint8(count))
		}
	}
}

// Bso обрабатывает получение БСО
func (ch *ChatBot) Bso(m *tgbotapi.Message) {
	log.Println("Bso", *m)
	bso, err := strconv.ParseUint(m.Text, 10, 32)
	msg := tgbotapi.NewMessage(m.Chat.ID, "")
	if err != nil {
		msg.Text = "Не удалось найти номер БСО, попробуйте еще раз"
		ch.bot.Send(msg)
		return
	}
	ch.state.SetBso(m.Chat.ID, uint32(bso))
	ch.state.SetAction(m.Chat.ID, "amount")
	msg.Text = "Сумма оказанных услуг"
	ch.bot.Send(msg)
}

// Amount обрабатывает получение суммы услуг
func (ch *ChatBot) Amount(m *tgbotapi.Message) {
	log.Println("Amount", *m)
	msg := tgbotapi.NewMessage(m.Chat.ID, "")
	amount, err := strconv.ParseUint(m.Text, 10, 16)
	if err != nil {
		msg.Text = "не удалось распознать сумму услуг"
		ch.bot.Send(msg)
		return
	}
	ch.state.SetAmount(m.Chat.ID, uint16(amount))
	ch.state.SetAction(m.Chat.ID, "services")
	ch.Services(&tgbotapi.CallbackQuery{From: m.From, Message: m})

}

// Comment последний рубеж, добавляет комментарии пользователя и отправляет координатору
func (ch *ChatBot) Comment(m *tgbotapi.Message) {
	log.Println("Comment", *m)
	ch.state.SetComment(m.Chat.ID, m.Text)
	rep := ch.state.MakeReport(m.Chat.ID)
	msg := tgbotapi.NewMessage(m.Chat.ID, rep)
	ch.bot.Send(msg)
	for _, chat := range ch.state.super {
		msg = tgbotapi.NewMessage(chat, rep)
		ch.bot.Send(msg)
	}
	msg = tgbotapi.NewMessage(-300011805, rep)
	ch.bot.Send(msg)
	ch.state.Clear(m.Chat.ID)
}

// Cancel отмена заполнение отчета
func (ch *ChatBot) Cancel(chatid int64) {
	log.Println("Cancel", chatid)
	ch.state.Clear(chatid)
}

// Soft обрабатывает софтовые работы
func (ch *ChatBot) Soft(cal *tgbotapi.CallbackQuery) {
	log.Println("Soft", *cal)
	id, err := strconv.ParseInt(cal.Data, 10, 32)
	if err == nil {
		ch.state.AddService(cal.Message.Chat.ID, &Service{Type: 0, Job: uint8(id)})
	}
	if cal.Data == "remove" {
		ch.state.SetAction(cal.Message.Chat.ID, "services")
		ch.Services(cal)
	}
}

// Cable обрабатывает кабельные работы
func (ch *ChatBot) Cable(cal *tgbotapi.CallbackQuery) {
	log.Println("Cable", *cal)
	id, err := strconv.ParseInt(cal.Data, 10, 32)
	if err == nil {
		ch.state.AddService(cal.Message.Chat.ID, &Service{Type: 1, Job: uint8(id)})
	}
	if cal.Data == "remove" {
		ch.state.SetAction(cal.Message.Chat.ID, "services")
		ch.Services(cal)
	}
}

// TV обрабатывает ТВ работы
func (ch *ChatBot) TV(cal *tgbotapi.CallbackQuery) {
	log.Println("TV", *cal)
	id, err := strconv.ParseInt(cal.Data, 10, 32)
	if err == nil {
		ch.state.AddService(cal.Message.Chat.ID, &Service{Type: 2, Job: uint8(id)})
	}
	if cal.Data == "remove" {
		ch.state.SetAction(cal.Message.Chat.ID, "services")
		ch.Services(cal)
	}
}

// Router обрабатывает Роутерные работы
func (ch *ChatBot) Router(cal *tgbotapi.CallbackQuery) {
	log.Println("Router", *cal)
	id, err := strconv.ParseInt(cal.Data, 10, 32)
	if err == nil {
		ch.state.AddService(cal.Message.Chat.ID, &Service{Type: 3, Job: uint8(id)})
	}
	if cal.Data == "remove" {
		ch.state.SetAction(cal.Message.Chat.ID, "services")
		ch.Services(cal)
	}
}

// Run запуск работы бота
func (ch *ChatBot) Run() {
	log.Println("Run")

	log.Printf("Authorized on account %s", ch.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := ch.bot.GetUpdatesChan(u)
	if err != nil {
		log.Println(err.Error())
	}

	for update := range updates {
		go ch.ParseUpdate(&update)
	}
}
