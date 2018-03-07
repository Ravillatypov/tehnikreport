package tehnikreport

import (
	"strings"

	"gopkg.in/telegram-bot-api.v4"
)

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

// функция отправляет сообщение-инструкцию как пользоваться ботом
func Help(b *tgbotapi.BotAPI, u *tgbotapi.Update) {
	msg := tgbotapi.NewMessage(u.Message.Chat.ID, `Бот предназначен для сбора отчетов о выполненных работах.
		для отправки отчетов необходимо авторизваться с помощью комманды /login
		после авторизации можно будет смотреть свои незакрытые заявки коммандой /tiket
		под каждой заявкой есть кнопочка, с помощю которой можно отправить отчет`)
	b.Send(msg)
	video := tgbotapi.NewVideoUpload(u.Message.Chat.ID, "help.avi")
	video.Caption = "пример как пользоваться"
	b.Send(video)
}

// парсинг сообщения для сбора нужной информации по отчету
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
