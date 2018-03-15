package tehnikreport

import (
	"crypto/sha512"
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/go-sql-driver/mysql" // у нас все работает на mysql
)

// Db тип для взаимодействия с базой СУЗа
// перед началом должен быть инициализирован
type Db struct {
	mysql           *sql.DB   // структура базы
	sUserByPhone    *sql.Stmt // найти пользователя по  номеру телефона
	uUserChatid     *sql.Stmt // изменение chat_id пользователя
	sTiketsByUserid *sql.Stmt // выборка незакрытых заявок пользователя по id
}

// Tiket тип для хранения заявки
type Tiket struct {
	ID              uint32
	Client, Address string
}

// Initialize функция для инициализации
func Initialize(dbconfig string) (*Db, error) {
	suz, err := sql.Open("mysql", dbconfig)
	if err != nil {
		return &Db{}, err
	}
	suphon, err := suz.Prepare(`SELECT id,status,password,fio FROM mms_adm_users WHERE phone_number LIKE ? LIMIT 1`)
	if err != nil {
		return &Db{}, err
	}
	uuchat, err := suz.Prepare(`UPDATE mms_adm_users SET chat_id=? WHERE id=? LIMIT 1`)
	if err != nil {
		return &Db{}, err
	}
	stbid, err := suz.Prepare(`SELECT id,client,address FROM suz_orders WHERE executor_id = ? AND (coordination = 2 OR coordination = 20)`)
	if err != nil {
		return &Db{}, err
	}
	return &Db{mysql: suz, sUserByPhone: suphon, uUserChatid: uuchat, sTiketsByUserid: stbid}, nil
}

// Login функция для авторизации техников
func (d *Db) Login(phone string, pass string, ChatID int64) (res bool, id uint16, fio string) {
	log.Println("DB Login", phone, pass, ChatID)
	ln := strings.Count(phone, "")
	if ln < 11 {
		return
	}
	phone = phone[ln-11:]
	var (
		passwd string
		status int
	)
	log.Println(phone)
	hash := sha512.New()
	hash.Write([]byte(pass))
	pass = fmt.Sprintf("%x", hash.Sum(nil))
	log.Println(pass)
	rows, err := d.sUserByPhone.Query("%" + phone)
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer rows.Close()
	rows.Next()
	err = rows.Scan(&id, &status, &passwd, &fio)
	if err != nil {
		log.Println("DB Login", err.Error())
		return
	}
	log.Printf("id = %d, status = %d", id, status)
	if status == 0 && id != 0 && pass == passwd {
		res = true
		rs, err := d.uUserChatid.Exec(ChatID, id)
		if err != nil {
			log.Println(err.Error())
			return
		}
		affect, err := rs.RowsAffected()
		log.Printf("updated %d rows", affect)
		return
	}
	return
}

// SuperLogin функция для авторизации техников
// авторизация по номеру телефона(без отправки смс)
func (d *Db) SuperLogin(phone, pass string, ChatID int64) (bool, uint16) {
	log.Println("DB SuperLogin", phone, ChatID)
	ln := strings.Count(phone, "")
	if ln < 11 {
		return false, 0
	}
	phone = phone[ln-11:]
	var (
		id     uint16
		passwd string
		status int
	)
	hash := sha512.New()
	hash.Write([]byte(pass))
	pass = fmt.Sprintf("%x", hash.Sum(nil))
	rows, err := d.mysql.Query("SELECT id,status,password FROM mms_adm_users WHERE gid != 12 AND phone_number LIKE ? LIMIT 1", "%"+phone)
	if err != nil {
		log.Println(err.Error())
		return false, 0
	}
	defer rows.Close()
	rows.Next()
	err = rows.Scan(&id, &status, &passwd)
	if err != nil {
		log.Println("DB Login", err.Error())
		return false, 0
	}
	log.Printf("id = %d, status = %d", id, status)
	if status == 0 && id != 0 && pass == passwd {
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
		if err != nil {
			log.Println("Db LoadTikets", uid, err.Error())
			return t
		}
		defer rows.Close()
		for rows.Next() {
			var (
				tiketid         uint32
				client, address string
			)
			err = rows.Scan(&tiketid, &client, &address)
			if err == nil {
				t = append(t, Tiket{ID: tiketid, Client: client, Address: address})
				log.Println("Db LoadTikets", uid, tiketid, client, address)
			} else {
				log.Println("Db LoadTikets", uid, err.Error())
			}
		}

	}
	return t
}

// LoadUsers загружает с базы авторизованные учетки
func (d *Db) LoadUsers() (map[int64]uint16, map[int64]string) {
	log.Println("Db LoadUsers")
	ids := make(map[int64]uint16)
	names := make(map[int64]string)
	var (
		chatid int64
		uid    uint16
		fio    string
	)
	rows, err := d.mysql.Query(`SELECT id,chat_id,fio FROM mms_adm_users WHERE chat_id != 0 AND status = 0 AND gid = 12`)
	if err != nil {
		log.Println("Db LoadUsers", err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&uid, &chatid, &fio)
		ids[chatid] = uid
		names[chatid] = fio
		log.Println("Db LoadUsers", uid, chatid, fio)
	}
	return ids, names
}

// LoadSupers загружает с базы авторизованные учетки
func (d *Db) LoadSupers() []int64 {
	res := make([]int64, 0)
	var chatid int64
	rows, err := d.mysql.Query(`SELECT chat_id FROM mms_adm_users WHERE chat_id != 0 AND status = 0 AND gid != 12`)
	if err != nil {
		log.Println("Db LoadSupers", err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&chatid)
		res = append(res, chatid)
		log.Println("LOadSuper: ", chatid)
	}
	log.Println("LOadSuper: ", res)
	return res
}
