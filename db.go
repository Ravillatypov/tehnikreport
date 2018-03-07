package tehnikreport

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/telegram-bot-api.v4"
)

// тип для взаимодействия с базой СУЗа
// перед началом должен быть инициализирован
type Db struct {
	mysql           *sql.DB   // структура базы
	sUserByPhone    *sql.Stmt // найти пользователя по  номеру телефона
	sUserByChatid   *sql.Stmt // найти пользователя по chat_id
	uUserChatid     *sql.Stmt // изменение chat_id пользователя
	sTiketsByUserid *sql.Stmt // выборка незакрытых заявок пользователя по id
}

func Initialize(dbconfig string) (*Db, error) {
	suz, err := sql.Open("mysql", dbconfig)
	defer suz.Close()
	if err != nil {
		return &Db{}, err
	}
	suphon, err := suz.Prepare(`SELECT id,status FROM mms_adm_users WHERE phone_number LIKE ? LIMIT 1`)
	if err != nil {
		return &Db{}, err
	}
	suchat, err := suz.Prepare(`SELECT id,status FROM mms_adm_users WHERE chat_id LIKE ? LIMIT 1`)
	if err != nil {
		return &Db{}, err
	}
	uuchat, err := suz.Prepare(`UPDATE mms_adm_users SET chat_id=? WHERE id=? LIMIT 1`)
	if err != nil {
		return &Db{}, err
	}
	stbid, err := suz.Prepare(`SELECT id,client,address FROM suz_orders WHERE executor_id = ? AND coordination = 2`)
	if err != nil {
		return &Db{}, err
	}
	return &Db{mysql: suz, sUserByChatid: suchat, sUserByPhone: suphon, uUserChatid: uuchat, sTiketsByUserid: stbid}, nil
}

// функция для авторизации техников
// авторизация по номеру телефона(без отправки смс)
func (d *Db) Login(b *tgbotapi.BotAPI, u *tgbotapi.Update) {
	ln := strings.Count(u.Message.Contact.PhoneNumber, "")
	if ln < 11 {
		b.Send(tgbotapi.NewMessage(u.Message.Chat.ID, "Номер слишком короткий"))
	}
	phone := u.Message.Contact.PhoneNumber[ln-11:]
	id, status := 5, 55
	err := d.sUserByPhone.QueryRow("%"+phone).Scan(&id, &status)
	CrashIfError(err)
	log.Printf("id = %d, status = %d", id, status)
	if status == 0 && id != 0 {
		rs, err := d.uUserChatid.Exec(u.Message.Chat.ID, id)
		CrashIfError(err)
		affect, err := rs.RowsAffected()
		CrashIfError(err)
		log.Printf("updated %d rows", affect)
		b.Send(tgbotapi.NewMessage(u.Message.Chat.ID, "Идентификация пройдена успешно!"))
	} else {
		b.Send(tgbotapi.NewMessage(u.Message.Chat.ID, "Извини дружище, я тебя не узнаю. Похоже, тебя нет в системе."))
	}
}

func CrashIfError(er error) {
	if er != nil {
		log.Panic(er.Error())
	}
}

// функция отправляет технику список незакрытых заявок
// для каждой заявки добавляется кнопка для отправки отчета
func (d *Db) Tiket(b *tgbotapi.BotAPI, u *tgbotapi.Update) {
	var (
		id, tiketid     int
		address, client string
	)
	err := d.sUserByChatid.QueryRow(u.Message.Chat.ID).Scan(&id)
	CrashIfError(err)
	if id != 0 {
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, "")
		rows, err := d.sTiketsByUserid.Query(id)
		if err != nil {
			b.Send(tgbotapi.NewMessage(u.Message.Chat.ID, "что-то пошло не так! не могу прочитать заявки"))
		} else {
			for rows.Next() {
				err = rows.Scan(&tiketid, &client, &address)
				if err != nil {
					msg.Text = fmt.Sprintf("%s\n%s", client, address)
					keyb := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(client, fmt.Sprint("r", tiketid))))
					msg.ReplyMarkup = keyb
					b.Send(msg)
				}
			}

		}
	} else {
		b.Send(tgbotapi.NewMessage(u.Message.Chat.ID, "для получения списка заявок необходимо авторизоваться"))
	}
}

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
			tikid, err := strconv.Atoi(strings.Split(calbackdata, "report")[1])
			if err != nil {
				log.Println(err.Error())
				return
			}
			Reports[u.CallbackQuery.Message.Chat.ID] = Report{Id: tikid}

		}
	}
}
