package main

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sashabaranov/go-openai"
	"log"
)

// ContextQueue представляет кольцевую очередь для хранения контекста разговора с пользователем
type ContextQueue struct {
	data   []string
	size   int
	head   int
	tail   int
	length int
}

// NewContextQueue создает новую кольцевую очередь указанного размера
func NewContextQueue(size int) *ContextQueue {
	return &ContextQueue{
		data:   make([]string, size),
		size:   size,
		head:   0,
		tail:   0,
		length: 0,
	}
}

// Push добавляет новый элемент в кольцевую очередь
func (q *ContextQueue) Push(item string) {
	if q.length == q.size {
		q.head = (q.head + 1) % q.size
	}
	q.data[q.tail] = item
	q.tail = (q.tail + 1) % q.size
	if q.length < q.size {
		q.length++
	}
}

// GetAll возвращает все элементы кольцевой очереди
func (q *ContextQueue) GetAll() string {
	result := "Контекст разговора:\n--Начало контекста--\n"
	for i := 0; i < q.length; i++ {
		result += q.data[(q.head+i)%q.size]
	}
	result += "\n--Конец контекста--\n"

	return result
}

func main() {
	// Токен для доступа к API Telegram
	token := "6423022834:AAHv8IOOkbQFJ-NxUmJi1oikDTYA_9vE8a4"

	// Токен для доступа к API OpenAI
	openAIKey := "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"

	// Настройка подключения к API Telegram
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	// Установка режима отладки
	bot.Debug = true

	log.Printf("Авторизован как %s", bot.Self.UserName)

	// Создание канала для получения обновлений от API Telegram
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := bot.GetUpdatesChan(updateConfig)
	// Создаем клиента для работы с API OpenAI
	client := openai.NewClient(openAIKey)

	ChatContext := NewContextQueue(10)
	// Обработка полученных сообщений
	for update := range updates {
		if update.Message == nil {
			continue
		}
		if update.Message.Command() == "start" {

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Привет! Я телеграм-бот с доступом к ChatGPT. Отправь мне текст, и я его обработаю.")
			bot.Send(msg)
			continue
		}
		// Обработка текстовых сообщений
		if update.Message.Text != "" {

			// Получаем котекст и собираем promt
			prompt := ""
			if ChatContext.length > 0 {
				prompt += ChatContext.GetAll()
			}
			prompt += update.Message.Text
			// Отправляем сообщение пользователю об ожидании ответа
			typing := tgbotapi.NewChatAction(update.Message.Chat.ID, tgbotapi.ChatTyping)
			bot.Send(typing)
			// Отправляем запрос к API OpenAI и получаем ответ
			response, err := client.CreateChatCompletion(
				context.Background(),
				openai.ChatCompletionRequest{
					Model: openai.GPT3Dot5Turbo,
					Messages: []openai.ChatCompletionMessage{
						{
							Role:    openai.ChatMessageRoleUser,
							Content: prompt,
						},
					},
				},
			)
			if err != nil {
				fmt.Printf("ChatCompletion error: %v\n", err)
				return
			}
			// Обновляем контекст
			ChatContext.Push(update.Message.Text)
			ChatContext.Push(response.Choices[0].Message.Content)
			//

			// Отправляем ответ пользователю
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, response.Choices[0].Message.Content)
			bot.Send(msg)
		}
	}
}
