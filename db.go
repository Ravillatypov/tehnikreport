package tehnikreport

import (
	"database/sql"
	"log"
	"strings"

	_ "github.com/go-sql-driver/mysql" // у нас все работает на mysql
)

// Db тип для взаимодействия с базой СУЗа
// перед началом должен быть инициализирован
type Db struct {
	mysql           *sql.DB   // структура базы
	sUserByPhone    *sql.Stmt // найти пользователя по  номеру телефона
	sUserByChatid   *sql.Stmt // найти пользователя по chat_id
	uUserChatid     *sql.Stmt // изменение chat_id пользователя
	sTiketsByUserid *sql.Stmt // выборка незакрытых заявок пользователя по id
}

// Tiket тип для хранения заявки
type Tiket struct {
	ID     uint32
	Client string
}

// Initialize функция для инициализации
func Initialize(dbconfig string) (*Db, error) {
	suz, err := sql.Open("mysql", dbconfig)
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
	stbid, err := suz.Prepare(`SELECT id,client FROM suz_orders WHERE executor_id = ? AND coordination = 2`)
	if err != nil {
		return &Db{}, err
	}
	return &Db{mysql: suz, sUserByChatid: suchat, sUserByPhone: suphon, uUserChatid: uuchat, sTiketsByUserid: stbid}, nil
}

// Login функция для авторизации техников
// авторизация по номеру телефона(без отправки смс)
func (d *Db) Login(phone string, ChatID int64) (bool, uint16) {
	log.Println("DB Login", phone, ChatID)
	ln := strings.Count(phone, "")
	if ln < 11 {
		return false, 0
	}
	phone = phone[ln-11:]
	status := 55
	var id uint16
	rows, err := d.sUserByPhone.Query("%" + phone)
	defer rows.Close()
	if err != nil {
		log.Println(err.Error())
		return false, 0
	}
	rows.Next()
	err = rows.Scan(&id, &status)
	if err != nil {
		log.Println("DB Login", err.Error())
		return false, 0
	}
	log.Printf("id = %d, status = %d", id, status)
	if status == 0 && id != 0 {
		rs, err := d.uUserChatid.Exec(ChatID, id)
		if err != nil {
			log.Println(err.Error())
			return true, id
		}
		affect, err := rs.RowsAffected()
		log.Printf("updated %d rows", affect)
		return true, id
	}
	return false, 0
}

// LoadTikets возвращает список незакрытых заявок
func (d *Db) LoadTikets(uid uint16) []Tiket {
	log.Println("Db LoadTikets", uid)
	t := make([]Tiket, 0)
	if uid != 0 {
		rows, err := d.sTiketsByUserid.Query(uid)
		defer rows.Close()
		if err != nil {
			log.Println("Db LoadTikets", uid, err.Error())
		}
		for rows.Next() {
			var (
				tiketid uint32
				client  string
			)
			err = rows.Scan(&tiketid, &client)
			if err == nil {
				t = append(t, Tiket{ID: tiketid, Client: client})
				log.Println("Db LoadTikets", uid, tiketid, client)
			} else {
				log.Println("Db LoadTikets", uid, err.Error())
			}
		}

	}
	return t
}

// LoadUsers загружает с базы авторизованные учетки
func (d *Db) LoadUsers() *map[int64]uint16 {
	log.Println("Db LoadUsers")
	res := make(map[int64]uint16)
	var chatid int64
	var uid uint16
	rows, err := d.mysql.Query(`SELECT id,chat_id FROM mms_adm_users WHERE chat_id != 0 AND status = 0`)
	defer rows.Close()
	if err != nil {
		log.Println("Db LoadUsers", err.Error())
	}
	for rows.Next() {
		rows.Scan(&uid, &chatid)
		res[chatid] = uid
		log.Println("Db LoadUsers", uid, chatid)
	}
	log.Println("Db LoadUsers", res)
	return &res
}

// LoadSupers загружает с базы авторизованные учетки
func (d *Db) LoadSupers() []int64 {
	res := make([]int64, 0)
	var chatid int64
	rows, err := d.mysql.Query(`SELECT chat_id FROM mms_adm_users WHERE chat_id != 0 AND status = 0 AND gid != 12`)
	defer rows.Close()
	if err != nil {
		log.Println("Db LoadSupers", err.Error())
	}
	for rows.Next() {
		rows.Scan(&chatid)
		res = append(res, chatid)
	}
	return res
}
