package tehnikreport

import "sync"

// Report хранит введенные данные техником
// после заполнения необходимых полей, отчет отправляется координатору
// струкура удаляется
type Report struct {
	Id, BSO   int        // номер заявки и номер БСО
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
			"Настройка шифрованconvия",
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

type ChatState struct {
	sync.RWMutex                   // нужна синхронизация для мапов
	reports      map[uint64]Report // сохраняем для формировании  отчета
	action       map[uint64]string // что ждем от пользователя, какую инфу
	users        map[uint64]int    // сопоставление chat_id и внутеннего id
}

func (c *ChatState) GetAction(u uint64) string {
	c.RLock()
	defer c.RUnlock()
	return c.action[u]
}

func (c *ChatState) SetAction(u uint64, ac string) {
	c.Lock()
	defer c.Unlock()
	c.action[u] = ac
}

func (c *ChatState) GetReport(u uint64) Report {
	c.RLock()
	defer c.RUnlock()
	return c.reports[u]
}

func (c *ChatState) AddService(u uint64, s *Service) {
	c.RLock()
	defer c.RUnlock()
	rep := c.reports[u]
	rep.Services = append(rep.Services, (*s))
	c.reports[u] = rep
}
func (c *ChatState) SetStatus(u uint64, s bool) {
	c.RLock()
	defer c.RUnlock()
	rep := c.reports[u]
	rep.Status = s
	c.reports[u] = rep
}
func (c *ChatState) GetStatus(u uint64) bool {
	c.RLock()
	defer c.RUnlock()
	rep := c.reports[u]
	return rep.Status
}
