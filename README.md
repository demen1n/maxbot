# Maxbot

[![Go Reference](https://pkg.go.dev/badge/github.com/demen1n/maxbot.svg)](https://pkg.go.dev/github.com/demen1n/maxbot)
[![Go Report Card](https://goreportcard.com/badge/github.com/demen1n/maxbot)](https://goreportcard.com/report/github.com/demen1n/maxbot)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Библиотека, вдохновлённая библиотекой [telebot](https://github.com/tucnak/telebot). Библиотека авторов Max [max-bot-api-client-go](https://github.com/max-messenger/max-bot-api-client-go) имеет фатальный недостаток — её писал не я.
Не для продакшна (not production ready). Пока что не оттестированно и не дописано.

## Ссылки

- 📖 [Официальная документация MAX API](https://dev.max.ru/docs-api)
- 🏛️ [Официальная библиотека MAX](https://github.com/mail-ru-im/bot-golang)
- 🤖 [telebot - вдохновение для этой библиотеки](https://github.com/tucnak/telebot)

## Возможности

- 🚀 Простой и понятный API
- 🔄 Long Polling и Webhook
- 🎨 Inline клавиатуры
- 📁 Отправка файлов (фото, видео, аудио, документы)
- 🛡️ Middleware система
- 👥 Поддержка групповых чатов
- 📊 Встроенные метрики
- ⚡ Готовые middleware (rate limiting, whitelist, и др.)

## Установка

```bash
go get github.com/demen1n/maxbot
```

## Быстрый старт

```go
package main

import (
    "log"
    "os"
    
    "github.com/demen1n/maxbot"
)

func main() {
    b, err := maxbot.NewBot(maxbot.Settings{
        Token: os.Getenv("MAX_BOT_TOKEN"),
    })
    if err != nil {
        log.Fatal(err)
    }
    
    b.Handle("/start", func(c maxbot.Context) error {
        return c.Send("👋 Привет!")
    })
    
    b.Start()
}
```

## Документация

### Создание бота

```go
b, err := maxbot.NewBot(maxbot.Settings{
    Token:  "your-bot-token",
    URL:    maxbot.DefaultAPIURL, // опционально
    Logger: log.Default(),         // опционально
    Poller: &maxbot.LongPoller{    // опционально
        Timeout: 30 * time.Second,
    },
    OnError: func(err error, c maxbot.Context) {
        // глобальный обработчик ошибок
    },
})
```

### Обработка команд

```go
// Простая команда
b.Handle("/start", func(c maxbot.Context) error {
    return c.Send("Привет!")
})

// Команда с аргументами
b.Handle("/echo", func(c maxbot.Context) error {
    return c.Send(c.Payload())
})

// Получение аргументов как слайс
b.Handle("/user", func(c maxbot.Context) error {
    args := c.Args() // /user John Doe -> ["John", "Doe"]
    if len(args) == 0 {
        return c.Send("Укажите имя пользователя")
    }
    return c.Send("Привет, " + args[0])
})
```

### Inline клавиатуры

```go
b.Handle("/menu", func(c maxbot.Context) error {
    menu := &maxbot.ReplyMarkup{}
    
    // Добавляем кнопки построчно
    menu.Row(
        menu.Data("Кнопка 1", "btn1"),
        menu.Data("Кнопка 2", "btn2"),
    )
    menu.Row(
        menu.URL("Открыть сайт", "https://example.com"),
    )
    
    return c.Send("Выберите действие:", menu)
})

// Обработка нажатий
menu := &maxbot.ReplyMarkup{}
b.Handle(&menu.Data("Кнопка 1", "btn1"), func(c maxbot.Context) error {
    return c.Send("Вы нажали кнопку 1")
})
```

### Отправка файлов

```go
// Фото
photo := &maxbot.Photo{FileID: "file_id"}
b.Send(user, photo, &maxbot.SendOptions{
    Text: "Подпись к фото",
})

// Документ
doc := &maxbot.Document{Token: "file_token"}
b.Send(user, doc)

// Загрузка файла
token, err := b.UploadFile("image", "photo.jpg", fileData)
```

### Редактирование сообщений

```go
b.Handle("/edit", func(c maxbot.Context) error {
    msg := c.Message()
    return c.Edit("Отредактированный текст")
})

// Редактирование с клавиатурой
menu := &maxbot.ReplyMarkup{}
menu.Row(menu.Data("Новая кнопка", "new"))
b.Edit(msg, "Новый текст", menu)
```

### Middleware

```go
import "github.com/demen1n/maxbot/middleware"

// Логирование
b.Handle("/start", handler, middleware.Logger())

// Whitelist пользователей
b.Handle("/admin", adminHandler, 
    middleware.Whitelist(123456789, 987654321))

// Rate limiting
b.Handle("/weather", weatherHandler,
    middleware.RateLimit(5, time.Minute))

// Только приватные чаты
b.Handle("/settings", settingsHandler,
    middleware.OnlyPrivate())

// Цепочка middleware
b.Handle("/cmd", handler,
    middleware.Chain(
        middleware.Logger(),
        middleware.AutoRespond(),
        middleware.Throttle(5 * time.Second),
    ))
```

### Доступные middleware

- `Logger()` - логирование запросов
- `AutoRespond()` - автоответ на callback queries
- `Recover()` - восстановление после паник
- `Whitelist(ids...)` - разрешить только указанным пользователям
- `Blacklist(ids...)` - заблокировать указанных пользователей
- `Throttle(duration)` - ограничение частоты использования
- `RateLimit(max, window)` - лимит запросов в окне времени
- `OnlyPrivate()` - только приватные чаты
- `OnlyGroups()` - только групповые чаты
- `IgnoreBots()` - игнорировать сообщения от ботов
- `CommandArgs(min, usage)` - проверка минимального числа аргументов
- `Chain(...)` - объединение нескольких middleware

### Webhook

```go
webhook := &maxbot.Webhook{
    Listen:   ":8443",
    Endpoint: "/webhook",
    URL:      "https://example.com/webhook",
    Secret:   "secret_key",
}

b, err := maxbot.NewBot(maxbot.Settings{
    Token:  token,
    Poller: webhook,
})

// Регистрация webhook в MAX API
b.SetWebhook(webhook.URL, []string{"message", "callback"}, webhook.Secret)
```

### Работа с чатами

```go
// Получить информацию о чате
chat, err := b.GetChat(chatID)

// Получить администраторов
admins, err := b.GetChatAdmins(chatID)

// Управление участниками
b.KickChatMember(chatID, userID)
b.InviteChatMembers(chatID, []int64{user1, user2})
b.PromoteChatMember(chatID, userID)
b.DemoteChatMember(chatID, userID)

// Закрепление сообщений
b.PinMessage(chatID, messageID)
b.UnpinMessage(chatID)

// Действия в чате (typing, отправка фото и т.д.)
b.SendChatAction(chatID, maxbot.ActionTyping)
```

### Context методы

```go
func handler(c maxbot.Context) error {
    // Информация об обновлении
    c.Bot()      // *Bot
    c.Update()   // Update
    c.Message()  // *Message
    c.Callback() // *CallbackQuery
    c.Sender()   // *User
    c.Chat()     // *Chat
    
    // Текст и аргументы
    c.Text()     // текст сообщения
    c.Args()     // аргументы команды как слайс
    c.Payload()  // всё после команды как строка
    
    // Отправка
    c.Send("текст", opts...)
    c.Reply("текст", opts...)
    c.Edit("новый текст", opts...)
    c.Delete()
    c.Respond() // ответ на callback
    
    // Хранилище
    c.Set("key", value)
    c.Get("key")
    
    return nil
}
```

### Метрики

```go
metrics := &middleware.Metrics{}

b.Handle("/start", handler, metrics.Middleware())

// Получить статистику
b.Handle("/stats", func(c maxbot.Context) error {
    return c.Send(metrics.GetStats())
})
```