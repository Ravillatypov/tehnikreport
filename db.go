package tehnikreport

import (
	"database/sql"
	"log"
	"strings"

	_ "github.com/go-sql-driver/mysql"
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

type Tiket struct {
	Id     int
	Client string
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
func (d *Db) Login(phone string, chat_id uint64) bool {
	ln := strings.Count(phone, "")
	if ln < 11 {
		return false
	}
	phone = phone[ln-11:]
	id, status := 5, 55
	err := d.sUserByPhone.QueryRow("%"+phone).Scan(&id, &status)
	CrashIfError(err)
	log.Printf("id = %d, status = %d", id, status)
	if status == 0 && id != 0 {
		rs, err := d.uUserChatid.Exec(chat_id, id)
		CrashIfError(err)
		affect, err := rs.RowsAffected()
		CrashIfError(err)
		log.Printf("updated %d rows", affect)
		return true
	}
	return false
}

// функция отправляет технику список незакрытых заявок
// для каждой заявки добавляется кнопка для отправки отчета
func (d *Db) LoadTikets(uid int) []Tiket {
	var (
		tiketid         int
		address, client string
	)
	t := make([]Tiket, 1)
	if uid != 0 {
		rows, err := d.sTiketsByUserid.Query(uid)
		if err == nil {
			for rows.Next() {
				err = rows.Scan(&tiketid, &client, &address)
				if err != nil {
					t = append(t, Tiket{Id: tiketid, Client: client})
				}
			}
		}
	}
	return t
}

func (d *Db) LoadUsers() map[uint64]int {
	res := make(map[uint64]int)
	var chatid uint64
	var uid int
	rows, err := d.mysql.Query(`SELECT id,chat_id FROM mms_adm_users WHERE chat_id != 0 AND status = 0`)
	if err == nil {
		for rows.Next() {
			rows.Scan(&uid, &chatid)
			res[chatid] = uid
		}
	}
	return res
}

func CrashIfError(er error) {
	if er != nil {
		log.Panic(er.Error())
	}
}
