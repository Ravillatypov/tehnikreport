package tehnikreport

import (
	"fmt"
	"log"
	"sync"

	"gopkg.in/telegram-bot-api.v4"
)

// Report хранит введенные данные техником
// после заполнения необходимых полей, отчет отправляется координатору
// струкура удаляется
type Report struct {
	ID, BSO   int        // номер заявки и номер БСО
	Comment   string     // комментарии техника по заявке, здесь же можно указать услуги
	Status    bool       // заявка выполнена или нет
	Services  []Service  // перечень выполненных работ
	Amount    float32    // сумма оказанных услуг
	Materials []Material // затраченные материалы
}

// Material id материала и количество
type Material struct {
	ID, Count int
}

// Service какая работа была выполнена
type Service struct {
	Type, Job int
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
			"Настройка шифрованconvия",
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
	ReportForm = `id заявки %d
	номер БСО: %d
	сумма: %f
	выполненные работы: %s
	%s
	комментарии: %s
	`
)

// ChatState тип для хранения состояния чата
type ChatState struct {
	sync.RWMutex                  // нужна синхронизация для мапов
	reports      map[int64]Report // сохраняем для формировании  отчета
	action       map[int64]string // что ждем от пользователя, какую инфу
	users        map[int64]int    // сопоставление chat_id и внутеннего id
	super        []int64
}

// GetAction используется для получения action
func (c *ChatState) GetAction(u int64) string {
	log.Println("GetAction", u)
	c.RLock()
	defer c.RUnlock()
	return c.action[u]
}

// SetAction задает следующее действие для чата
func (c *ChatState) SetAction(u int64, ac string) {
	log.Println("SetAction", u, ac)
	c.Lock()
	defer c.Unlock()
	a := c.action
	log.Println(a)
	a[u] = ac
	c.action = a
}

// GetReport формирует отчет координатору
func (c *ChatState) GetReport(u int64) string {
	log.Println("GetReport", u)
	c.RLock()
	defer c.RUnlock()
	r := c.reports[u]
	log.Println(r)
	return r.MakeReport()
}

// AddService добавляет выполненную работу
func (c *ChatState) AddService(u int64, s *Service) {
	log.Println("AddService", u, *s)
	c.Lock()
	defer c.Unlock()
	rep := c.reports[u]
	for _, v := range rep.Services {
		if v == *s {
			return
		}
	}
	rep.Services = append(rep.Services, (*s))
	c.reports[u] = rep
}

// AddMaterials добавляет выполненную работу
func (c *ChatState) AddMaterials(u int64, m *Material) {
	log.Println("AddMaterials", u, *m)
	c.Lock()
	defer c.Unlock()
	rep := c.reports[u]
	newmat := make([]Material, 0)
	newmat = append(newmat, *m)
	for _, v := range rep.Materials {
		if v == *m {
			return
		}
		if v.Count == 0 {
			continue
		}
		newmat = append(newmat, v)
	}
	rep.Materials = newmat
	c.reports[u] = rep
}

// SetMaterialsCount добавляет выполненную работу
func (c *ChatState) SetMaterialsCount(u int64, count int) {
	log.Println("SetMaterialCount", u, count)
	c.Lock()
	defer c.Unlock()
	rep := c.reports[u]

	for k, v := range rep.Materials {
		if v.Count == 0 {
			rep.Materials[k].Count = count
			return
		}
	}
	c.reports[u] = rep
}

// SetStatus устанавливает статус выполнения заявки
func (c *ChatState) SetStatus(u int64, s bool) {
	log.Println("SetStatus", u, s)
	c.Lock()
	defer c.Unlock()
	rep := c.reports[u]
	rep.Status = s
	c.reports[u] = rep
}

// GetStatus получает статус выполнения заявки
func (c *ChatState) GetStatus(u int64) bool {
	log.Println("GetStatus", u)
	c.RLock()
	defer c.RUnlock()
	rep := c.reports[u]
	return rep.Status
}

// MakeReport создает отчет координатору
func (r *Report) MakeReport() string {
	log.Println("MakeReport")
	if !r.Status {
		return fmt.Sprintf("заявка с id = %d не выполнена", r.ID)
	}
	var allservises string
	materials := "Материалы: "
	for _, i := range r.Services {
		allservises += i.Print()
	}
	for _, m := range r.Materials {
		materials += m.Print()
	}
	if r.Amount >= 1000.0 {
		return fmt.Sprintf(ReportForm, r.ID, r.BSO, r.Amount,
			"Выезд;\n"+allservises,
			materials, r.Comment)
	}
	return fmt.Sprintf(ReportForm, r.ID, r.BSO, r.Amount,
		"Выезд;\nДиагностика;\n"+allservises,
		materials, r.Comment)

}

// Print формирует строку для печати
func (s *Service) Print() string {
	return ServiceList[s.Type][s.Job] + ";\n"
}

// Print формирует строку для печати
func (m *Material) Print() string {
	if m.ID == 5 || m.ID == 6 {
		return MaterialList[m.ID] + fmt.Sprintf(" %d м.;\n", m.Count)
	}
	return MaterialList[m.ID] + fmt.Sprintf(" %d шт.;\n", m.Count)
}

// AddUser добавляет нового пользователя
func (c *ChatState) AddUser(chatid int64, uid int) {
	log.Println("AddUser", chatid, uid)
	c.Lock()
	defer c.Unlock()
	c.users[chatid] = uid
}

// GetUserID получаем id пользователя
func (c *ChatState) GetUserID(chatid int64) int {
	log.Println("GetUserID", chatid)
	c.RLock()
	defer c.RUnlock()
	return c.users[chatid]
}

// IsCable были ли кабельные работы
func (c *ChatState) IsCable(chatid int64) bool {
	c.RLock()
	defer c.RUnlock()
	for _, s := range c.reports[chatid].Services {
		if s.Type == 1 {
			return true
		}
	}
	return false
}

// AddReport создаем новый отчет для данного чата
func (c *ChatState) AddReport(chatid int64, reportid int) {
	c.Lock()
	defer c.Unlock()
	c.reports[chatid] = Report{ID: reportid}
}

// SetBso установливаем номер БСО
func (c *ChatState) SetBso(chatid int64, bso int) {
	c.Lock()
	defer c.Unlock()
	r := c.reports[chatid]
	r.BSO = bso
	c.reports[chatid] = r
}

// SetAmount установливаем сумму услуг
func (c *ChatState) SetAmount(chatid int64, amount float32) {
	c.Lock()
	defer c.Unlock()
	r := c.reports[chatid]
	r.Amount = amount
	c.reports[chatid] = r
}

// Clear очищает отчет, состояние чата
func (c *ChatState) Clear(chatid int64) {
	c.Lock()
	defer c.Unlock()
	c.reports[chatid] = Report{}
	c.SetAction(chatid, "")
}

// SetComment меняем коммент
func (c *ChatState) SetComment(chatid int64, comment string) {
	c.Lock()
	defer c.Unlock()
	r := c.reports[chatid]
	r.Comment = comment
	c.reports[chatid] = r
}

// MakeReport
func (c *ChatState) MakeReport(chatid int64) string {
	c.RLock()
	defer c.RUnlock()
	r := c.reports[chatid]
	return r.MakeReport()
}

// LoadUsers меняем коммент
func (c *ChatState) LoadUsers(uids *map[int64]int) {
	log.Println("ChatState LoadUsers", *uids)
	c.Lock()
	defer c.Unlock()
	c.users = (*uids)
}

// AddSuper меняем коммент
func (c *ChatState) AddSuper(chatid int64) {
	c.Lock()
	defer c.Unlock()
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
