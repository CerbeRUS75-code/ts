Архитектура системы
Компоненты системы
AI-агент (NLP Processor)
Отвечает за понимание запросов пользователей. Использует простую модель обработки естественного языка (NLP) на основе ключевых слов для определения интента запроса.
Особенности: Легковесный и stateless, что позволяет запускать несколько экземпляров для параллельной обработки запросов.
Масштабируемость: Подходит для горизонтального масштабирования с увеличением числа обработчиков.
База данных (Knowledge Base)
Хранит предопределенные ответы для каждого интента в виде структуры данных (например, map[Intent]string).
Особенности: In-memory реализация с потокобезопасным доступом через механизмы вроде sync.RWMutex для параллельного чтения.
Масштабируемость: Может быть заменена на внешнюю базу данных (например, PostgreSQL или Redis) для работы с большими объемами данных.
Система очередей (Support Agent и AI Support System)
Управляет обработкой запросов в реальном времени.
Support Agent: Координирует запросы, направляя их либо к AI-агенту, либо к человеку через буферизованный канал (humanQueue, вместимостью 50 запросов).
AI Support System: Управляет пулом воркеров (например, горутин в Go), которые обрабатывают запросы из канала (queries, вместимостью 100 запросов).
Масштабируемость: Буферизованные каналы и пул воркеров обеспечивают конкурентную обработку множества запросов, а ограничение размера очередей предотвращает перегрузку.
Обеспечение масштабируемости
Использование конкурентных механизмов для параллельной обработки запросов.
Ограничение размера очередей для управления нагрузкой.
Возможность горизонтального масштабирования путем добавления воркеров или интеграции с балансировщиками нагрузки (например, Nginx или Kubernetes).

Ниже приведен фрагмент кода, который реализует простую систему обработки запросов, включая определение интента с помощью базовой NLP-модели и запрос к имитационной базе данных. Этот код соответствует требованиям задания (1-2 страницы) и основан на предоставленной структуре.

package main

import (
	"strings"
	"sync"
)

// Intent представляет намерение пользователя
type Intent string

const (
	IntentGreeting Intent = "greeting"
	IntentQuestion Intent = "question"
	IntentUnknown  Intent = "unknown"
)

// Query — структура запроса пользователя
type Query struct {
	ID   int
	Text string
}

// Response — структура ответа
type Response struct {
	QueryID int
	Text    string
	Source  string
}

// NLPProcessor определяет интент запроса
type NLPProcessor struct {
	keywords map[string]Intent
}

func NewNLPProcessor() *NLPProcessor {
	return &NLPProcessor{
		keywords: map[string]Intent{
			"привет": IntentGreeting,
			"вопрос": IntentQuestion,
		},
	}
}

func (p *NLPProcessor) DetectIntent(text string) Intent {
	text = strings.ToLower(text)
	for keyword, intent := range p.keywords {
		if strings.Contains(text, keyword) {
			return intent
		}
	}
	return IntentUnknown
}

// KnowledgeBase — имитация базы знаний
type KnowledgeBase struct {
	answers map[Intent]string
	mu      sync.RWMutex
}

func NewKnowledgeBase() *KnowledgeBase {
	return &KnowledgeBase{
		answers: map[Intent]string{
			IntentGreeting: "Здравствуйте! Чем могу помочь?",
			IntentQuestion: "Задайте свой вопрос, и я постараюсь ответить.",
		},
	}
}

func (kb *KnowledgeBase) GetAnswer(intent Intent) string {
	kb.mu.RLock()
	defer kb.mu.RUnlock()
	if answer, exists := kb.answers[intent]; exists {
		return answer
	}
	return "Извините, я не могу ответить на этот запрос."
}

// SupportAgent обрабатывает запросы
type SupportAgent struct {
	nlp *NLPProcessor
	kb  *KnowledgeBase
}

func NewSupportAgent(nlp *NLPProcessor, kb *KnowledgeBase) *SupportAgent {
	return &SupportAgent{nlp: nlp, kb: kb}
}

func (a *SupportAgent) ProcessQuery(query Query) Response {
	intent := a.nlp.DetectIntent(query.Text)
	if intent == IntentUnknown {
		return Response{QueryID: query.ID, Text: "Перенаправляю ваш запрос человеку.", Source: "human"}
	}
	answer := a.kb.GetAnswer(intent)
	return Response{QueryID: query.ID, Text: answer, Source: "ai"}
}

func main() {
	nlp := NewNLPProcessor()
	kb := NewKnowledgeBase()
	agent := NewSupportAgent(nlp, kb)

	query := Query{ID: 1, Text: "Привет, как дела?"}
	response := agent.ProcessQuery(query)
	println(response.Text) // Вывод: "Здравствуйте! Чем могу помочь?"
}

