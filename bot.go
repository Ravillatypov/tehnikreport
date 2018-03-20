package tehnikreport

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"io"
	"net/http"
	"os"

	"gopkg.in/telegram-bot-api.v4"
)

// ChatBot тип для хране2ния всего что нужно в одном месте
type ChatBot struct {
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
	u, n := d.LoadUsers()
	s := &ChatState{
		reports: make(map[int64]*Report),
		super:   d.LoadSupers(),
		action:  String{s: make(map[int64]string)},
		phone:   String{s: make(map[int64]string)},
		name:    String{s: n},
		users:   Uint16{s: u},
	}
	b, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Println(err.Error())
		return new(ChatBot), err
	}
	return &ChatBot{db: d, bot: b, state: s}, nil
}

// Sendmsg отправляет сообщение, проверяет ошибки
func (ch *ChatBot) Sendmsg(m tgbotapi.MessageConfig) {
	log.Println("Send message", m)
	_, err := ch.bot.Send(m)
	if err != nil {
		log.Println("Send message error: ", err.Error())
	}
}

// Help функция отправляет сообщение-инструкцию как пользоваться ботом
func (ch *ChatBot) Help(m *tgbotapi.Message) {
	log.Println("Help", *m)
	msg := tgbotapi.NewMessage(m.Chat.ID, `Бот предназначен для сбора отчетов о выполненных работах.
для отправки отчетов необходимо авторизваться с помощью комманд /login или сокращенно /l
после авторизации можно будет смотреть свои незакрытые заявки коммандами /tiket или /t
под каждой заявкой есть кнопочка, с помощю которой можно отправить отчет
комманды /super и /s для авторизации как руководитель или как координатор.
/cancel или /c отменяет заполнение отчета.
комманды /help и /h отправляют данную справку

по всем вопросам и пожеланиями можно писать @Ravil_Latypov`)
	ch.Sendmsg(msg)
	video := tgbotapi.NewVideoUpload(m.Chat.ID, "help.mp4")
	video.Caption = "пример"
	ch.Sendmsg(msg)
}

// ParseUpdate по состоянию чата пользователя раздает задачи функциям
func (ch *ChatBot) ParseUpdate(u *tgbotapi.Update) {
	log.Println("ParseUpdate", *u)
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
		return
	}
	var chat_id int64
	if u.CallbackQuery != nil {
		chat_id = u.CallbackQuery.Message.Chat.ID
	}
	if u.Message != nil {
		chat_id = u.Message.Chat.ID
	}
	switch ch.state.action.get(chat_id) {
	case "refuse":
		go ch.Refuse(u)
	case "transfer":
		go ch.Transfer(u)
	case "done":
		go ch.Done(u)
	case "beneficial":
		go ch.Beneficial(u)
	case "photo":
		go ch.Photo(u)
	case "date":
		go ch.Date(u)
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
	case "send":
		go ch.Send(u.CallbackQuery)
	case "bso":
		go ch.Bso(u.Message)
	case "amount":
		go ch.Amount(u.Message)
	case "comment":
		go ch.Comment(u.Message)
	case "login":
		go ch.Login(u.Message)
	case "super":
		go ch.Super(u.Message)
	default:
		go ch.NewReport(u.CallbackQuery)
	}
}

// DefaultParse парсит CallbackQuery
func (ch *ChatBot) DefaultParse(u *tgbotapi.Update) {
	if u.CallbackQuery == nil {
		return
	}
	dat := u.CallbackQuery.Data
	switch {
	case strings.HasPrefix(dat, "report"):
		ch.NewReport(u.CallbackQuery)
	case strings.HasPrefix(dat, "refuse"):
		ch.Refuse(u)
	case strings.HasPrefix(dat, "beneficial"):
		ch.Beneficial(u)
	case strings.HasPrefix(dat, "transfer"):
		ch.Transfer(u)
	}
}

// Refuse обработка отказа
func (ch *ChatBot) Refuse(u *tgbotapi.Update) {
}

// Transfer обработка перенос заявки
func (ch *ChatBot) Transfer(u *tgbotapi.Update) {
}

// Done обработка выполненной работы
func (ch *ChatBot) Done(u *tgbotapi.Update) {
}

// Beneficial обработка льготной заявки
func (ch *ChatBot) Beneficial(u *tgbotapi.Update) {
}

// Photo сохранение фото по льготной заявки
func (ch *ChatBot) Photo(u *tgbotapi.Update) {
	if u.Message.Photo != nil {
		for _, ph := range *u.Message.Photo {
			name := fmt.Sprintf("/var/www/suz.iqivision.pro/web/uploads/%d.jpg", time.Now().UnixNano())
			fd, err := os.Create(name)
			if err != nil {
				log.Println(err.Error())
				continue
			}
			defer fd.Close()
			tl := tgbotapi.File{FileID: ph.FileID, FileSize: ph.FileSize}
			link := tl.Link(ch.bot.Token)
			resp, err := http.Get(link)
			if err != nil {
				io.Copy(fd, resp.Body)
			}
			defer resp.Body.Close()
		}
	}
}

// Date установка даты переноса
func (ch *ChatBot) Date(u *tgbotapi.Update) {
	if u.Message != nil {
		now := time.Now()
		tm := now
		err := tm.UnmarshalText([]byte(u.Message.Text))
		if err != nil || tm.Unix() < now.Unix() {
			log.Println("Date: дата указана не правильно")
			return
		}
	}
}

// Tiket получает список заявок пользователя и отправляет с кнопкой для отчета
func (ch *ChatBot) Tiket(m *tgbotapi.Message) {
	log.Println("Tiket", *m)
	msg := tgbotapi.NewMessage(m.Chat.ID, "нет открытых заявок")
	uid := ch.state.users.get(m.Chat.ID)
	tikets := ch.db.LoadTikets(uid)
	log.Println("Tiket", uid, tikets)
	if len(tikets) > 0 {
		for _, t := range tikets {
			msg.Text = fmt.Sprintf("клиент: %s\nадрес: %s", t.Client, t.Address)

			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("выполнено", fmt.Sprintf("report%d", t.ID)),
					tgbotapi.NewInlineKeyboardButtonData("перенос", fmt.Sprintf("transfer%d", t.ID)),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("льготная", fmt.Sprintf("beneficial%d", t.ID)),
					tgbotapi.NewInlineKeyboardButtonData("отказ", fmt.Sprintf("refuse%d", t.ID)),
				))
			ch.Sendmsg(msg)
		}
		return
	}
	ch.Sendmsg(msg)
}

// Login авторизация
func (ch *ChatBot) Login(m *tgbotapi.Message) {
	log.Println("Login", *m)
	msg := tgbotapi.NewMessage(m.Chat.ID, "")
	switch {
	case m.IsCommand():
		log.Println("Login is command")
		msg.Text = "отправьте свой номер для авторизации"
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButtonContact("мой номер")))
		ch.state.action.set(m.Chat.ID, "login")
	case m.Contact != nil:
		log.Println("Login is contact", m.Contact.PhoneNumber)
		ln := strings.Count(m.Contact.PhoneNumber, "")
		if ln < 11 {
			msg.Text = "Номер телефона слишком короткий"
			ch.Sendmsg(msg)
			return
		}
		ch.state.phone.set(m.Chat.ID, m.Contact.PhoneNumber)
		msg.Text = "введите пароль"
		msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		ch.Sendmsg(msg)
		return
	case strings.Count(m.Text, "") > 0:
		log.Println("Login is password", m.Text)
		log.Println(ch.db)
		if stat, uid, fio := ch.db.Login(ch.state.phone.get(m.Chat.ID), m.Text, m.Chat.ID); stat {
			msg.Text = "Аутентификация пройдена успешно"
			log.Println(ch.state.action.get(m.Chat.ID))
			ch.state.action.del(m.Chat.ID)
			ch.state.users.set(m.Chat.ID, uid)
			ch.state.name.set(m.Chat.ID, fio)
		} else {
			msg.Text = "не правильный номер или пароль"
		}
	}
	ch.Sendmsg(msg)
}

// Super авторизация
func (ch *ChatBot) Super(m *tgbotapi.Message) {
	log.Println("Super", *m)
	msg := tgbotapi.NewMessage(m.Chat.ID, "")
	switch {
	case m.IsCommand():
		msg.Text = "отправьте свой номер для авторизации"
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButtonContact("мой номер")))
		ch.state.action.set(m.Chat.ID, "super")
	case m.Contact != nil:
		ln := strings.Count(m.Contact.PhoneNumber, "")
		if ln < 11 {
			msg.Text = "Номер телефона слишком короткий"
			return
		}
		ch.state.phone.set(m.Chat.ID, m.Contact.PhoneNumber)
		msg.Text = "введите пароль"
		msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		ch.Sendmsg(msg)
		return
	case strings.Count(m.Text, "") > 0:
		if stat, _ := ch.db.SuperLogin(ch.state.phone.get(m.Chat.ID), m.Text, m.Chat.ID); stat {
			msg.Text = "Аутентификация пройдена успешно"
			ch.state.AddSuper(m.Chat.ID)
			ch.state.Clear(m.Chat.ID)
		} else {
			msg.Text = "не правильный номер или пароль"
			ch.state.Clear(m.Chat.ID)
		}
	}
	ch.Sendmsg(msg)
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
		ch.state.action.set(cal.Message.Chat.ID, "bso")
		ch.state.reports[cal.Message.Chat.ID] = &Report{ID: uint32(id), Status: true}
		ch.bot.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: cal.Message.Chat.ID, MessageID: cal.Message.MessageID})
		ch.Sendmsg(tgbotapi.NewMessage(cal.Message.Chat.ID, "Введите номер БСО"))
	}
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
				ch.state.action.set(u.CallbackQuery.Message.Chat.ID, "materials")
				msg.Text = "какие материалы были использованы"
				msg.ReplyMarkup = GetMaterialsKeyb(ch.state.reports[u.CallbackQuery.Message.Chat.ID])
			} else {
				ch.state.action.set(u.CallbackQuery.Message.Chat.ID, "comment")
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
		ch.Sendmsg(msg)
	}
	if u.Message != nil {
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, "")
		ch.state.reports[u.Message.Chat.ID].DopServices = u.Message.Text
		if ch.state.IsCable(u.Message.Chat.ID) {
			ch.state.action.set(u.Message.Chat.ID, "materials")
			msg.Text = "какие материалы были использованы"
			msg.ReplyMarkup = GetMaterialsKeyb(ch.state.reports[u.Message.Chat.ID])
		} else {
			ch.state.action.set(u.Message.Chat.ID, "comment")
			msg.Text = "ваши комментарии к заявке"
		}
		ch.Sendmsg(msg)
	}

}

// Services получает списой выполненных работ
func (ch *ChatBot) Services(cal *tgbotapi.CallbackQuery) {
	log.Println("Services", *cal)
	if ch.state.action.get(cal.Message.Chat.ID) != "services" {
		return
	}
	msg := tgbotapi.NewMessage(cal.Message.Chat.ID, "выберите выполненные работы (множественный выбор)")
	ch.bot.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: cal.Message.Chat.ID, MessageID: cal.Message.MessageID})
	switch cal.Data {
	case "0":
		ch.state.action.set(cal.Message.Chat.ID, "soft")
		msg.ReplyMarkup = GetKeyboard(0, ch.state.reports[cal.Message.Chat.ID])
	case "1":
		ch.state.action.set(cal.Message.Chat.ID, "cable")
		msg.ReplyMarkup = GetKeyboard(1, ch.state.reports[cal.Message.Chat.ID])
	case "2":
		ch.state.action.set(cal.Message.Chat.ID, "tv")
		msg.ReplyMarkup = GetKeyboard(2, ch.state.reports[cal.Message.Chat.ID])
	case "3":
		ch.state.action.set(cal.Message.Chat.ID, "router")
		msg.ReplyMarkup = GetKeyboard(3, ch.state.reports[cal.Message.Chat.ID])
	case "return":
		ch.state.action.set(cal.Message.Chat.ID, "dopservices")
		ch.DopServices(&tgbotapi.Update{CallbackQuery: cal})
		return
	default:
		//ch.state.action.set(cal.Message.Chat.ID, "soft")
		msg.ReplyMarkup = ServiceTypeKeyb
	}
	ch.Sendmsg(msg)
}

// Materials получает список материалов
func (ch *ChatBot) Materials(u *tgbotapi.Update) {
	log.Println("Materials", *u)
	if u.CallbackQuery != nil {
		id, err := strconv.ParseUint(u.CallbackQuery.Data, 10, 32)
		if err == nil {
			ch.state.AddMaterials(u.CallbackQuery.Message.Chat.ID, &Material{ID: uint8(id)})
			ch.Sendmsg(tgbotapi.NewMessage(u.CallbackQuery.Message.Chat.ID, "количество?"))
			return
		}
		if u.CallbackQuery.Data == "remove" {
			ch.bot.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: u.CallbackQuery.Message.Chat.ID, MessageID: u.CallbackQuery.Message.MessageID})
			ch.state.action.set(u.CallbackQuery.Message.Chat.ID, "comment")
			msg := tgbotapi.NewMessage(u.CallbackQuery.Message.Chat.ID, "Ваши комментари к заявке")
			ch.Sendmsg(msg)
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
	if err != nil || bso < 10000 || bso > 999999 {
		msg.Text = "Не удалось найти номер БСО, попробуйте еще раз"
		ch.Sendmsg(msg)
		return
	}
	ch.state.reports[m.Chat.ID].BSO = uint32(bso)
	ch.state.action.set(m.Chat.ID, "amount")
	msg.Text = "Сумма оказанных услуг"
	ch.Sendmsg(msg)
}

// Amount обрабатывает получение суммы услуг
func (ch *ChatBot) Amount(m *tgbotapi.Message) {
	log.Println("Amount", *m)
	msg := tgbotapi.NewMessage(m.Chat.ID, "")
	amount, err := strconv.ParseUint(m.Text, 10, 16)
	if err != nil {
		msg.Text = "не удалось распознать сумму услуг"
		ch.Sendmsg(msg)
		return
	}
	ch.state.reports[m.Chat.ID].Amount = uint16(amount)
	ch.state.action.set(m.Chat.ID, "services")
	ch.Services(&tgbotapi.CallbackQuery{From: m.From, Message: m})

}

// Send отправка отчета
func (ch *ChatBot) Send(cal *tgbotapi.CallbackQuery) {
	ch.bot.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: cal.Message.Chat.ID, MessageID: cal.Message.MessageID})
	switch cal.Data {
	case "true":
		log.Println(ch.state.super)
		txt := fmt.Sprintf(ForCoordirantors,
			ch.state.name.get(cal.Message.Chat.ID),
			ch.state.reports[cal.Message.Chat.ID].MakeReport())
		log.Println("Send super: ", ch.state.super)
		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("перейти к заявке",
					fmt.Sprintf("http://suz.iqvision.pro/view/orders/%d", ch.state.reports[cal.Message.Chat.ID].ID)),
			),
		)
		for _, chat := range ch.state.super {
			msg := tgbotapi.NewMessage(chat, txt)
			msg.ReplyMarkup = kb
			ch.Sendmsg(msg)
		}
		msg := tgbotapi.NewMessage(-300011805, txt)
		msg.ReplyMarkup = kb
		ch.Sendmsg(msg)
		ch.Sendmsg(tgbotapi.NewMessage(cal.Message.Chat.ID, "отчет отправлен"))
		ch.state.Clear(cal.Message.Chat.ID)

	case "false":
		ch.state.Clear(cal.Message.Chat.ID)
		ch.Tiket(cal.Message)
	}
}

// Comment последний рубеж, добавляет комментарии пользователя и отправляет координатору
func (ch *ChatBot) Comment(m *tgbotapi.Message) {
	log.Println("Comment", *m)
	ch.state.reports[m.Chat.ID].Comment = m.Text
	rep := ch.state.reports[m.Chat.ID].MakeReport()
	msg := tgbotapi.NewMessage(m.Chat.ID, rep)
	ch.Sendmsg(msg)
	msg.Text = "отправить отчет?"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("да", "true"),
			tgbotapi.NewInlineKeyboardButtonData("нет", "false"),
		),
	)
	ch.Sendmsg(msg)
	ch.state.action.set(m.Chat.ID, "send")
}

// Cancel отмена заполнение отчета
func (ch *ChatBot) Cancel(chatid int64) {
	log.Println("Cancel", chatid)
	ch.state.Clear(chatid)
	msg := tgbotapi.NewMessage(chatid, "заполнение отчета отменено")
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	ch.Sendmsg(msg)
}

// Soft обрабатывает софтовые работы
func (ch *ChatBot) Soft(cal *tgbotapi.CallbackQuery) {
	log.Println("Soft", *cal)
	id, err := strconv.ParseUint(cal.Data, 10, 8)
	if err == nil {
		ch.state.AddService(cal.Message.Chat.ID, &Service{Type: 0, Job: uint8(id)})
	}
	if cal.Data == "remove" {
		ch.state.action.set(cal.Message.Chat.ID, "services")
		ch.Services(cal)
	}
}

// Cable обрабатывает кабельные работы
func (ch *ChatBot) Cable(cal *tgbotapi.CallbackQuery) {
	log.Println("Cable", *cal)
	id, err := strconv.ParseUint(cal.Data, 10, 8)
	if err == nil {
		ch.state.AddService(cal.Message.Chat.ID, &Service{Type: 1, Job: uint8(id)})
	}
	if cal.Data == "remove" {
		ch.state.action.set(cal.Message.Chat.ID, "services")
		ch.Services(cal)
	}
}

// TV обрабатывает ТВ работы
func (ch *ChatBot) TV(cal *tgbotapi.CallbackQuery) {
	log.Println("TV", *cal)
	id, err := strconv.ParseUint(cal.Data, 10, 8)
	if err == nil {
		ch.state.AddService(cal.Message.Chat.ID, &Service{Type: 2, Job: uint8(id)})
	}
	if cal.Data == "remove" {
		ch.state.action.set(cal.Message.Chat.ID, "services")
		ch.Services(cal)
	}
}

// Router обрабатывает Роутерные работы
func (ch *ChatBot) Router(cal *tgbotapi.CallbackQuery) {
	log.Println("Router", *cal)
	id, err := strconv.ParseUint(cal.Data, 10, 8)
	if err == nil {
		ch.state.AddService(cal.Message.Chat.ID, &Service{Type: 3, Job: uint8(id)})
	}
	if cal.Data == "remove" {
		ch.state.action.set(cal.Message.Chat.ID, "services")
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
		if update.Message == nil && update.CallbackQuery == nil {
			continue
		}
		log.Println(update)
		go ch.ParseUpdate(&update)
	}
}
