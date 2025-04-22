package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// Основные типы интентов
type Intent string

const (
	IntentGreeting  Intent = "greeting"
	IntentHelp      Intent = "help"
	IntentPricing   Intent = "pricing"
	IntentTechnical Intent = "technical"
	IntentBilling   Intent = "billing"
	IntentUnknown   Intent = "unknown"
)

// Структуры для запросов и ответов
type Query struct {
	ID     string
	UserID string
	Text   string
}

type Response struct {
	QueryID string
	Text    string
	Source  string // "ai" или "human"
}

// Упрощенная база знаний
type KnowledgeBase struct {
	answers map[Intent]string
	mu      sync.RWMutex
}

func NewKnowledgeBase() *KnowledgeBase {
	kb := &KnowledgeBase{
		answers: map[Intent]string{
			IntentGreeting:  "Здравствуйте! Чем я могу вам помочь?",
			IntentHelp:      "Я могу помочь вам с вопросами о ценах, технической поддержке или счетах.",
			IntentPricing:   "Наш базовый тариф стоит $10/мес, премиум - $25/мес.",
			IntentTechnical: "Для технических вопросов уточните, с какой функцией у вас проблемы.",
			IntentBilling:   "По вопросам счетов обратитесь в финансовый отдел.",
		},
	}
	return kb
}

func (kb *KnowledgeBase) GetAnswer(intent Intent) string {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	if answer, ok := kb.answers[intent]; ok {
		return answer
	}
	return "Извините, я не могу ответить на этот вопрос."
}

// Упрощенный NLP процессор
type NLPProcessor struct {
	keywords map[string]Intent
}

func NewNLPProcessor() *NLPProcessor {
	return &NLPProcessor{
		keywords: map[string]Intent{
			"привет":       IntentGreeting,
			"здравствуй":   IntentGreeting,
			"здравствуйте": IntentGreeting,
			"добрый день":  IntentGreeting,
			"доброе утро":  IntentGreeting,
			"добрый вечер": IntentGreeting,

			"помощь":    IntentHelp,
			"помоги":    IntentHelp,
			"помогите":  IntentHelp,
			"поддержка": IntentHelp,

			"цена":      IntentPricing,
			"стоимость": IntentPricing,
			"тариф":     IntentPricing,
			"стоит":     IntentPricing,
			"план":      IntentPricing,

			"проблема":    IntentTechnical,
			"ошибка":      IntentTechnical,
			"не работает": IntentTechnical,
			"сломалось":   IntentTechnical,
			"техническая": IntentTechnical,
			"технический": IntentTechnical,
			"баг":         IntentTechnical,

			"счет":   IntentBilling,
			"оплата": IntentBilling,
			"счёт":   IntentBilling,
			"платеж": IntentBilling,
			"платёж": IntentBilling,
			"деньги": IntentBilling,
		},
	}
}

func (p *NLPProcessor) DetectIntent(text string) Intent {
	text = strings.ToLower(text)
	log.Printf("Определяем интент для запроса: %s", text)

	for keyword, intent := range p.keywords {
		if strings.Contains(text, keyword) {
			log.Printf("Найдено ключевое слово '%s', определен интент: %s", keyword, intent)
			return intent
		}
	}

	log.Printf("Интент не определен, запрос будет направлен оператору")
	return IntentUnknown
}

// Агент поддержки
type SupportAgent struct {
	nlp             *NLPProcessor
	kb              *KnowledgeBase
	humanQueue      chan Query
	maxHumanQueries int
}

func NewSupportAgent(nlp *NLPProcessor, kb *KnowledgeBase) *SupportAgent {
	return &SupportAgent{
		nlp:             nlp,
		kb:              kb,
		humanQueue:      make(chan Query, 50),
		maxHumanQueries: 50,
	}
}

func (a *SupportAgent) ProcessQuery(query Query) Response {
	intent := a.nlp.DetectIntent(query.Text)

	// Если интент неизвестен, передаем человеку
	if intent == IntentUnknown {
		select {
		case a.humanQueue <- query:
			log.Printf("Запрос %s передан человеку-оператору", query.ID)
			return Response{
				QueryID: query.ID,
				Text:    "Ваш запрос передан специалисту поддержки.",
				Source:  "human",
			}
		default:
			log.Printf("Очередь к оператору заполнена, отправляем стандартный ответ")
			return Response{
				QueryID: query.ID,
				Text:    "Все операторы заняты. Попробуйте переформулировать вопрос.",
				Source:  "ai",
			}
		}
	}

	// Возвращаем ответ из базы знаний
	answer := a.kb.GetAnswer(intent)
	log.Printf("На запрос %s найден ответ по интенту %s: %s", query.ID, intent, answer)

	return Response{
		QueryID: query.ID,
		Text:    answer,
		Source:  "ai",
	}
}

// Запускаем обработчик для человека-оператора
func (a *SupportAgent) StartHumanWorker(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case query := <-a.humanQueue:
				log.Printf("Человек обрабатывает запрос: %s", query.ID)
				// Имитация обработки
				time.Sleep(500 * time.Millisecond)
			}
		}
	}()
}

// Система обработки запросов с поддержкой конкурентности
type AISupport struct {
	agent      *SupportAgent
	workers    int
	queries    chan Query
	responses  map[string]chan Response
	responseMu sync.Mutex
}

func NewAISupport(agent *SupportAgent, workers int) *AISupport {
	return &AISupport{
		agent:     agent,
		workers:   workers,
		queries:   make(chan Query, 100),
		responses: make(map[string]chan Response),
	}
}

func (s *AISupport) Start(ctx context.Context) {
	// Запускаем обработчик для оператора
	s.agent.StartHumanWorker(ctx)

	// Запускаем воркеры для обработки запросов
	for i := 0; i < s.workers; i++ {
		go s.worker(ctx, i)
	}
}

func (s *AISupport) worker(ctx context.Context, id int) {
	log.Printf("Запущен воркер %d", id)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Воркер %d завершает работу", id)
			return
		case query := <-s.queries:
			log.Printf("Воркер %d обрабатывает запрос: %s", id, query.ID)

			// Обрабатываем запрос
			response := s.agent.ProcessQuery(query)

			// Отправляем ответ
			s.responseMu.Lock()
			if ch, ok := s.responses[query.ID]; ok {
				ch <- response
				delete(s.responses, query.ID)
			} else {
				log.Printf("Предупреждение: канал для ответа %s не найден", query.ID)
			}
			s.responseMu.Unlock()
		}
	}
}

// Обработка запроса с таймаутом
func (s *AISupport) Process(query Query, timeout time.Duration) (Response, error) {
	// Создаем канал для ответа
	respChan := make(chan Response, 1)

	// Регистрируем канал для получения ответа
	s.responseMu.Lock()
	s.responses[query.ID] = respChan
	s.responseMu.Unlock()

	// Отправляем запрос на обработку
	select {
	case s.queries <- query:
		log.Printf("Запрос %s добавлен в очередь обработки", query.ID)
	default:
		s.responseMu.Lock()
		delete(s.responses, query.ID)
		s.responseMu.Unlock()
		return Response{}, fmt.Errorf("система перегружена")
	}

	// Ожидаем ответ с таймаутом
	select {
	case resp := <-respChan:
		log.Printf("Получен ответ на запрос %s от %s", query.ID, resp.Source)
		return resp, nil
	case <-time.After(timeout):
		log.Printf("Таймаут ожидания ответа на запрос %s", query.ID)
		s.responseMu.Lock()
		delete(s.responses, query.ID)
		s.responseMu.Unlock()
		return Response{}, fmt.Errorf("таймаут ожидания ответа")
	}
}

func main() {
	// Инициализируем компоненты
	kb := NewKnowledgeBase()
	nlp := NewNLPProcessor()
	agent := NewSupportAgent(nlp, kb)
	support := NewAISupport(agent, 5) // 5 параллельных обработчиков

	// Запускаем систему
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	support.Start(ctx)

	// Примеры запросов
	queries := []Query{
		{ID: "q1", UserID: "u1", Text: "Привет, как дела?"},
		{ID: "q2", UserID: "u2", Text: "Сколько стоит ваш сервис?"},
		{ID: "q3", UserID: "u3", Text: "У меня не работает авторизация"},
		{ID: "q4", UserID: "u4", Text: "Как мне скачать отчет по транзакциям?"},
	}

	fmt.Println("===== Начало обработки запросов =====")

	// Обрабатываем запросы
	for _, q := range queries {
		fmt.Printf("\nОбработка запроса: %s\n", q.Text)
		resp, err := support.Process(q, 3*time.Second)
		if err != nil {
			log.Printf("Ошибка: %v", err)
			continue
		}

		fmt.Printf("Запрос: %s\nОтвет: %s\nИсточник: %s\n", q.Text, resp.Text, resp.Source)
	}

	fmt.Println("\n===== Завершение обработки запросов =====")

	// Даем время на завершение обработки
	time.Sleep(1 * time.Second)
}
