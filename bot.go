package tehnikreport

import (
	"strings"

	"gopkg.in/telegram-bot-api.v4"
)

// chatbot тип для хранения всего что нужно в одном месте
type chatbot struct {
	db    *Db
	state *ChatState
	bot   *tgbotapi.BotAPI
}

// переменная хранит значения введенные техником, но еще не отправленные координатору
// после отправки отчета коорднатору, удаляется элемент из карты
// ключем является chat_id
// кнопки выбора типа выполненных работ
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
func (ch *chatbot) Help(u *tgbotapi.Update) {
	msg := tgbotapi.NewMessage(u.Message.Chat.ID, `Бот предназначен для сбора отчетов о выполненных работах.
		для отправки отчетов необходимо авторизваться с помощью комманды /login
		после авторизации можно будет смотреть свои незакрытые заявки коммандой /tiket
		под каждой заявкой есть кнопочка, с помощю которой можно отправить отчет`)
	ch.bot.Send(msg)
	video := tgbotapi.NewVideoUpload(u.Message.Chat.ID, "help.avi")
	video.Caption = "пример как пользоваться"
	ch.bot.Send(video)
}

// ParseReport парсинг сообщения для сбора нужной информации по отчету
func ParseReport(b *tgbotapi.BotAPI, u *tgbotapi.Update) {
	if u.CallbackQuery != nil {
		var calbackdata = u.CallbackQuery.Data
		if strings.HasPrefix(calbackdata, "report") {
			b.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: u.CallbackQuery.Message.Chat.ID, MessageID: u.CallbackQuery.Message.MessageID})
			//tikid, err := strconv.Atoi(strings.Split(calbackdata, "report")[1])
			//if err != nil {
			//	log.Println(err.Error())
			//	return
			//}
		}
	}
}

// ParseUpdate
func (ch *chatbot) ParseUpdate(u *tgbotapi.Update) {}

// NewReport
func (ch *chatbot) NewReport(cal *tgbotapi.CallbackQuery) {}

// Status
func (ch *chatbot) Status(cal *tgbotapi.CallbackQuery) {}

// Services
func (ch *chatbot) Services(cal *tgbotapi.CallbackQuery) {}

// Materials
func (ch *chatbot) Materials(cal *tgbotapi.CallbackQuery) {}

// Bso
func (ch *chatbot) Bso(u *tgbotapi.Update) {}

// Amount
func (ch *chatbot) Amount(u *tgbotapi.Update) {}
