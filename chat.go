package tehnikreport

import (
	"fmt"
	"log"
	"sync"

	"gopkg.in/telegram-bot-api.v4"
)

// String map[int64]string
type String struct {
	sync.RWMutex
	s map[int64]string
}

func (s *String) set(id int64, val string) {
	s.Lock()
	defer s.Unlock()
	s.s[id] = val
}

func (s *String) del(id int64) {
	s.Lock()
	defer s.Unlock()
	delete(s.s, id)
}

func (s *String) get(id int64) string {
	s.RLock()
	defer s.RUnlock()
	return s.s[id]
}

// Uint16 map[int64]uint16
type Uint16 struct {
	sync.RWMutex
	s map[int64]uint16
}

func (s *Uint16) set(id int64, val uint16) {
	s.Lock()
	defer s.Unlock()
	s.s[id] = val
}

func (s *Uint16) del(id int64) {
	s.Lock()
	defer s.Unlock()
	delete(s.s, id)
}

func (s *Uint16) get(id int64) uint16 {
	s.RLock()
	defer s.RUnlock()
	return s.s[id]
}

// Report хранит введенные данные техником
// после заполнения необходимых полей, отчет отправляется координатору,
// в группу и пользователям авторизованных как super
// струкура удаляется
type Report struct {
	ID, BSO     uint32     // номер заявки и номер БСО
	Comment     string     // комментарии техника по заявке, здесь же можно указать услуги
	Status      bool       // заявка выполнена или нет
	Services    []Service  // перечень выполненных работ
	Amount      uint16     // сумма оказанных услуг
	Materials   []Material // затраченные материалы
	DopServices string     // дополнительные услуги
}

// Material id материала и количество
type Material struct {
	ID, Count uint8
}

// Service какая работа была выполнена
type Service struct {
	Type, Job uint8
}

var (
	// ServiceTypes типы выполняемых работ
	ServiceTypes = []string{
		"Софт",
		"Кабель",
		"Телевидение",
		"Роутер",
	}
	// ServiceList варианты выполняемых работ
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
	// MaterialList используемые материалы
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
	// ReportForm шаблон отчета
	ReportForm = `id заявки: %d
	номер БСО: %s
	сумма: %d
	выполненные работы: %s
	комментарии: %s
	`
	// ForCoordirantors шаблон письма для координатора
	ForCoordirantors = `Техник: %s
%s`
)

// ChatState тип для хранения состояния чата
type ChatState struct {
	reports map[int64]*Report // сохраняем для формировании  отчета
	action  String            // что ждем от пользователя, какую инфу
	phone   String            // номер телефона
	name    String            // ФИО в системе
	users   Uint16            // сопоставление chat_id и внутеннего id
	super   []int64           // чаты руководителей и коорднаторов
}

// AddService добавляет выполненную работу
func (c *ChatState) AddService(u int64, s *Service) {
	log.Println("AddService", u, *s)
	serv := c.reports.get(u).Services
	for _, v := range serv {
		if v == *s {
			return
		}
	}
	c.reports.get(u).Services = append(serv, (*s))
}

// AddMaterials добавляет материал
func (c *ChatState) AddMaterials(u int64, m *Material) {
	log.Println("AddMaterials", u, *m)
	newmat := make([]Material, 0)
	newmat = append(newmat, *m)
	for _, v := range c.reports.get(u).Materials {
		if v == *m {
			return
		}
		if v.Count == 0 {
			continue
		}
		newmat = append(newmat, v)
	}
	c.reports.get(u).Materials = newmat
}

// SetMaterialsCount указывает количество материала
func (c *ChatState) SetMaterialsCount(u int64, count uint8) {
	log.Println("SetMaterialCount", u, count)
	mat := make([]Material, 0)
	for _, v := range c.reports.get(u).Materials {
		if v.Count == 0 {
			v.Count = count
			log.Printf("SetCount %v", mat)
		}
		mat = append(mat, v)
	}
	c.reports.get(u).Materials = mat
}

// MakeReport создает отчет координатору
func (r *Report) MakeReport() string {
	log.Println("MakeReport")
	bso := fmt.Sprintf("%d", r.BSO)
	if r.BSO < 100000 {
		bso = fmt.Sprintf("0%d", r.BSO)
	}
	if !r.Status {
		return fmt.Sprintf(`id заявки: %d
			заявка не выполнена
			комментарии: %s`, r.ID, r.Comment)
	}
	var allservises string
	materials := "\nМатериалы: \n"
	for _, i := range r.Services {
		allservises += i.Print()
	}
	if r.DopServices != "" {
		allservises += "\nДополнительные услуги:\n" + r.DopServices
	}
	for _, m := range r.Materials {
		materials += m.Print()
	}
	if materials == "\nМатериалы: \n" {
		materials = ""
	}
	if r.Amount >= 1000 {
		return fmt.Sprintf(ReportForm, r.ID, bso, r.Amount,
			"\nВыезд;\n"+allservises+materials, r.Comment)
	}
	return fmt.Sprintf(ReportForm, r.ID, bso, r.Amount,
		"\nВыезд;\nДиагностика;\n"+allservises+materials, r.Comment)

}

// Print формирует строку для печати
func (s *Service) Print() string {
	return ServiceList[s.Type][s.Job] + ";\n"
}

// Print формирует строку для печати
func (m *Material) Print() string {
	if m.Count == 0 {
		return ""
	}
	if m.ID == 5 || m.ID == 6 {
		return MaterialList[m.ID] + fmt.Sprintf(" %d м.;\n", m.Count)
	}
	return MaterialList[m.ID] + fmt.Sprintf(" %d шт.;\n", m.Count)
}

// IsCable были ли кабельные работы
func (c *ChatState) IsCable(chatid int64) bool {
	for _, s := range c.reports.get(chatid).Services {
		if s.Type == 1 {
			return true
		}
	}
	return false
}

// Clear очищает отчет, состояние чата
func (c *ChatState) Clear(chatid int64) {
	c.reports.del(chatid)
	c.action.del(chatid)
}

// AddSuper меняем коммент
func (c *ChatState) AddSuper(chatid int64) {
	s := make([]int64, 0)
	for _, v := range c.super {
		if v == chatid {
			return
		}
		s = append(s, chatid)
	}
	c.super = s
}

// GetKeyboard создаем кнопки выбора услуг
func GetKeyboard(i int) *tgbotapi.InlineKeyboardMarkup {
	rows := make([][]tgbotapi.InlineKeyboardButton, 0)
	for k, v := range ServiceList[i] {
		rows = append(rows, []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(v, fmt.Sprintf("%d", k))})
	}
	rows = append(rows, []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("все введено", "remove")})
	return &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// GetMaterialsKeyb создаем кнопки выбора услуг
func GetMaterialsKeyb() *tgbotapi.InlineKeyboardMarkup {
	rows := make([][]tgbotapi.InlineKeyboardButton, 0)
	for k, v := range MaterialList {
		if k == 0 {
			continue
		}
		rows = append(rows, []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(v, fmt.Sprintf("%d", k))})
	}
	rows = append(rows, []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("все введено", "remove")})
	return &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}
