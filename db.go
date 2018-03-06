package tehnikreport

import (
	"database/sql"
	"log"
	"strings"

	"errors"

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/telegram-bot-api.v4"
)

// Report хранит введенные данные техником
// после заполнения необходимых полей, отчет отправляется координатору
// струкура удаляется
type Report struct {
	Id, BSO   int32      // номер заявки и номер БСО
	Comment   string     // комментарии техника по заявке, здесь же можно указать услуги
	Status    bool       // заявка выполнена или нет
	Services  []Service  // перечень выполненных работ
	Amount    float32    // сумма оказанных услуг
	Materials []Material // затраченные материалы
}

// id материала и количество
type Material struct {
	Id, Count int
}

// какая работа была выполнена
type Service struct {
	Type, Job int
}

// тип для взаимодействия с базой СУЗа
// перед началом должен быть инициализирован
type Db struct {
	mysql           *sql.DB   // структура базы
	sUserByPhone    *sql.Stmt // найти пользователя по  номеру телефона
	sUserByChatid   *sql.Stmt // найти пользователя по chat_id
	uUserChatid     *sql.Stmt // изменение chat_id пользователя
	sTiketsByUserid *sql.Stmt // выборка незакрытых заявок пользователя по id
}

var (
	ServiceTypes = []string{
		"Софт",
		"Кабель",
		"Телевидение",
		"Роутер",
	}
	ServiceList = [][]string{
		{
			"Оптимизация ОС",
			"Восстановление ОС",
			"Сканирование на вирусы",
			"Удаление вирусов",
			"Установка антивируса",
			"Чистка реестра",
			"Настройка PPPoE соединения",
			" Регистрация/восстановление УЗ",
			"Установка базового пакета ПО",
			"Установка офисного пакета",
			"Установка драйверов 1-10шт",
		},
		{
			"Укладка кабеля открытым способом",
			"Укладка кабеля закрытым способом",
			"Скочлок соединение",
			"Обжимка коннектора",
			"Установка F-разъема",
			"Установка ТВ-делителя",
			"Установка соединительной бочки",
			"Установка ТВ-штекера",
			"Строительно-монтажные работы",
		},
		{
			"Автоматическая настройка ТВ",
			"Ручная настройка каналов (1-60шт)",
			"Настройка SMART",
			"Обновление прошивки",
			"Сброс до заводских настроек",
			"Настройка цветности",
			"Настройка звука",
		},
		{
			"Настройка PPPoE",
			"Настройка канала вещания WiFi",
			"Настройка шифрования",
			"Обновление прошивки",
		},
	}
	MaterialList = []string{
		"",
		"Розетка",
		"Скотчлок",
		"Бочка",
		"RJ-45",
		"UTP-5",
		"RG-6",
		"ТВ-штеккер",
		"F-разъём",
		"Делитель ТВ",
	}
)

// переменная хранит значения введенные техником, но еще не отправленные координатору
// после отправки отчета коорднатору, удаляется элемент из карты
// ключем является chat_id
var Reports = make(map[uint64]Report)

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

// функция для авторизации техников
// авторизация по номеру телефона(без отправки смс)
func Login(b *tgbotapi.BotAPI, u *tgbotapi.Update) error {
	ln := strings.Count(u.Message.Contact.PhoneNumber, "")
	if ln < 11 {
		b.Send(tgbotapi.NewMessage(u.Message.Chat.ID, "Номер слишком короткий"))
		return errors.New("Номер слишком короткий")
	}
	phone := u.Message.Contact.PhoneNumber[ln-11:]
	log.Printf("phone: %s", phone)
	CrashIfError(err)
	log.Printf("db connected")
	id, status := 5, 55
	err = db.QueryRow("SELECT id,status FROM mms_adm_users WHERE phone_number LIKE ? LIMIT 1", "%"+phone).Scan(&id, &status)
	CrashIfError(err)
	log.Printf("id = %d, status = %d", id, status)
	if status == 0 && id != 0 {
		rs, err := db.Exec("UPDATE mms_adm_users SET chat_id=? WHERE id=? LIMIT 1", chat_id, id)
		CrashIfError(err)
		affect, err := rs.RowsAffected()
		CrashIfError(err)
		log.Printf("updated %d rows", affect)
		return "Идентификация пройдена успешно!", true
	} else {
		return "Извини дружище, я тебя не узнаю. Похоже, тебя нет в системе.", false
	}
}

func CrashIfError(er error) {
	if er != nil {
		log.Panic(er.Error())
	}
}

// функция отправляет технику список незакрытых заявок
// для каждой заявки добавляется кнопка для отправки отчета
func Tiket(b *tgbotapi.BotAPI, u *tgbotapi.Update) error {

}

// функция отправляет сообщение-инструкцию как пользоваться ботом
func Help(b *tgbotapi.BotAPI, u *tgbotapi.Update) error {

}

// парсинг сообщения для сбора нужной информации по отчету
func ParseReport(b *tgbotapi.BotAPI, u *tgbotapi.Update) error {

}
