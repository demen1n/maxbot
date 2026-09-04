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

// Обработка нажатий — сохраните кнопку в переменную перед передачей в Handle
btn1 := menu.Data("Кнопка 1", "btn1")
b.Handle(&btn1, func(c maxbot.Context) error {
    return c.Send("Вы нажали кнопку 1")
})
```

### Типы кнопок

```go
menu := &maxbot.ReplyMarkup{}

// Callback — отправляет payload боту
menu.Row(menu.Data("Нажми меня", "my_action"))

// Ссылка
menu.Row(menu.URL("Открыть сайт", "https://example.com"))

// Запрос контакта / геолокации
menu.Row(menu.Contact("Поделиться номером"))
menu.Row(menu.Geolocation("Поделиться локацией", false))

// Мини-приложение
menu.Row(menu.OpenApp("Открыть", "https://app.example.com", "deeplink", 0))

// Clipboard — копирует текст в буфер обмена
menu.Row(menu.Clipboard("Скопировать", "текст для копирования"))

// Создание чата
menu.Row(menu.Chat("Создать группу", "Моя группа", "Описание", "start"))

// Кнопка отправки шаблонного сообщения
menu.Row(menu.MessageBtn("Отправить"))
```

### Отправка файлов

Файлы сначала загружаются на серверы MAX, затем отправляются сообщением.

```go
// Фото
tokens, err := b.UploadPhoto("photo.jpg", fileData)
if err != nil {
    return err
}
b.Send(chat, &maxbot.Photo{PhotoTokens: *tokens}, &maxbot.SendOptions{
    Text: "Подпись к фото",
})

// Аудио, видео, файл
info, err := b.UploadMedia("file", "document.pdf", fileData)
if err != nil {
    return err
}
b.Send(chat, &maxbot.Document{UploadedInfo: *info})

// UploadFile — устаревший враппер, оставлен для совместимости
token, err := b.UploadFile("image", "photo.jpg", fileData)
```

### Редактирование сообщений

`Bot.Edit` и `Context.Edit` возвращают только `error` (не `*Message`).

```go
b.Handle("/edit", func(c maxbot.Context) error {
    return c.Edit("Отредактированный текст")
})

// Прямое редактирование через бот
if err := b.Edit(msg, "Новый текст"); err != nil {
    log.Println(err)
}
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

### Типы обновлений

```go
// Нажатие кнопки "Начать"
b.Handle(maxbot.OnBotStarted, func(c maxbot.Context) error {
    return c.Send("Добро пожаловать! deeplink: " + c.Update().Payload)
})

// Бот добавлен в чат / удалён из чата
b.Handle(maxbot.OnBotAdded, func(c maxbot.Context) error {
    return c.Send("Привет, " + c.Chat().Title + "!")
})
b.Handle(maxbot.OnBotRemoved, func(c maxbot.Context) error { return nil })

// Вступление и выход участников
b.Handle(maxbot.OnUserAdded, func(c maxbot.Context) error {
    return c.Send("Добро пожаловать, " + c.Sender().Name + "!")
})
b.Handle(maxbot.OnUserRemoved, func(c maxbot.Context) error { return nil })

// Изменение названия чата
b.Handle(maxbot.OnChatTitleChanged, func(c maxbot.Context) error {
    return c.Send("Чат переименован: " + c.Update().Title)
})

// Редактирование сообщения
b.Handle(maxbot.OnMessageEdited, func(c maxbot.Context) error {
    return nil
})

// Удаление сообщения
b.Handle(maxbot.OnMessageRemoved, func(c maxbot.Context) error {
    return nil
})
```

### Webhook

```go
webhook := &maxbot.Webhook{
    Listen:   ":8443",
    Endpoint: "/webhook",
    URL:      "https://example.com/webhook",
    Secret:   "secret_key", // проверяется через X-Max-Bot-Api-Secret
}

b, err := maxbot.NewBot(maxbot.Settings{
    Token:  token,
    Poller: webhook,
})

// Регистрация webhook в MAX API
b.SetWebhook(webhook.URL, []string{"message_created", "message_callback"}, webhook.Secret)

// Удаление webhook (URL обязателен)
b.DeleteWebhook("https://example.com/webhook")
```

### Работа с чатами

```go
// Получить информацию о чате
chat, err := b.GetChat(chatID)
chat, err := b.GetChatByLink("mygroup")

// Список чатов с пагинацией.
// Deprecated: с июня 2026 GET /chats больше не поддерживается MAX API.
// Замены на стороне API нет — собирайте chat_id сами из входящих апдейтов
// (bot_added, bot_started, message_created и т.д.) и храните их в своей БД.
chats, nextMarker, err := b.GetChats(50, nil)

// Участники с пагинацией
members, nextMarker, err := b.GetChatMembers(chatID, 100, nil)

// Получить администраторов
admins, marker, err := b.GetChatAdmins(chatID)

// Управление участниками
b.KickChatMember(chatID, userID, false)       // block=false — просто удалить
b.KickChatMember(chatID, userID, true)        // block=true — забанить
b.InviteChatMembers(chatID, []int64{user1, user2})
b.PromoteChatMember(chatID, userID)           // все права по умолчанию
b.PromoteChatMember(chatID, userID, maxbot.PermWrite, maxbot.PermPinMessage)
b.DemoteChatMember(chatID, userID)

// Закрепление сообщений
b.PinMessage(chatID, messageID, nil)          // notify=nil — серверное значение по умолчанию
notify := true
b.PinMessage(chatID, messageID, &notify)
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