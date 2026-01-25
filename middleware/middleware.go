// Package middleware provides common middleware implementations for maxbot.
package middleware

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/demen1n/maxbot"
)

// Logger logs all incoming updates with timing information.
func Logger() maxbot.MiddlewareFunc {
	return func(next maxbot.HandlerFunc) maxbot.HandlerFunc {
		return func(c maxbot.Context) error {
			start := time.Now()

			user := c.Sender()
			text := c.Text()

			if text != "" {
				log.Printf("[%s] @%s: %s", user.Name, user.Username, text)
			} else if cb := c.Callback(); cb != nil {
				log.Printf("[%s] @%s: callback(%s)", user.Name, user.Username, cb.Payload)
			}

			err := next(c)
			log.Printf("Processed in %v", time.Since(start))

			return err
		}
	}
}

// AutoRespond automatically responds to callback queries.
func AutoRespond() maxbot.MiddlewareFunc {
	return func(next maxbot.HandlerFunc) maxbot.HandlerFunc {
		return func(c maxbot.Context) error {
			if c.Callback() != nil {
				defer c.Respond()
			}
			return next(c)
		}
	}
}

// Recover recovers from panics in handlers.
func Recover(onPanic ...func(error)) maxbot.MiddlewareFunc {
	return func(next maxbot.HandlerFunc) maxbot.HandlerFunc {
		return func(c maxbot.Context) error {
			defer func() {
				if r := recover(); r != nil {
					err := fmt.Errorf("panic recovered: %v", r)
					log.Printf("PANIC: %v", err)

					if len(onPanic) > 0 && onPanic[0] != nil {
						onPanic[0](err)
					}

					c.Send("❌ Произошла внутренняя ошибка")
				}
			}()

			return next(c)
		}
	}
}

// Whitelist allows only specified users.
func Whitelist(userIDs ...int64) maxbot.MiddlewareFunc {
	allowed := make(map[int64]bool)
	for _, id := range userIDs {
		allowed[id] = true
	}

	return func(next maxbot.HandlerFunc) maxbot.HandlerFunc {
		return func(c maxbot.Context) error {
			if !allowed[c.Sender().ID] {
				return c.Send("⛔ У вас нет доступа к этой команде")
			}
			return next(c)
		}
	}
}

// Blacklist blocks specified users.
func Blacklist(userIDs ...int64) maxbot.MiddlewareFunc {
	blocked := make(map[int64]bool)
	for _, id := range userIDs {
		blocked[id] = true
	}

	return func(next maxbot.HandlerFunc) maxbot.HandlerFunc {
		return func(c maxbot.Context) error {
			if blocked[c.Sender().ID] {
				return nil
			}
			return next(c)
		}
	}
}

// Throttle limits handler execution to once per duration per user.
func Throttle(d time.Duration) maxbot.MiddlewareFunc {
	type key struct {
		userID  int64
		handler string
	}

	lastExec := make(map[key]time.Time)

	return func(next maxbot.HandlerFunc) maxbot.HandlerFunc {
		return func(c maxbot.Context) error {
			k := key{
				userID:  c.Sender().ID,
				handler: fmt.Sprintf("%p", next),
			}

			if last, ok := lastExec[k]; ok {
				if time.Since(last) < d {
					remaining := d - time.Since(last)
					return c.Send(fmt.Sprintf(
						"⏱ Подождите еще %v перед повторным использованием",
						remaining.Round(time.Second),
					))
				}
			}

			lastExec[k] = time.Now()
			return next(c)
		}
	}
}

// RestrictChatType restricts handler to specific chat types.
func RestrictChatType(types ...string) maxbot.MiddlewareFunc {
	allowed := make(map[string]bool)
	for _, t := range types {
		allowed[t] = true
	}

	return func(next maxbot.HandlerFunc) maxbot.HandlerFunc {
		return func(c maxbot.Context) error {
			chat := c.Chat()
			if chat == nil || !allowed[chat.Type] {
				return c.Send("⛔ Эта команда недоступна в данном типе чата")
			}
			return next(c)
		}
	}
}

// OnlyPrivate allows handler only in private chats.
func OnlyPrivate() maxbot.MiddlewareFunc {
	return RestrictChatType("private", "dialog")
}

// OnlyGroups allows handler only in group chats.
func OnlyGroups() maxbot.MiddlewareFunc {
	return RestrictChatType("group", "channel")
}

// CommandArgs ensures command has minimum number of arguments.
func CommandArgs(min int, usage string) maxbot.MiddlewareFunc {
	return func(next maxbot.HandlerFunc) maxbot.HandlerFunc {
		return func(c maxbot.Context) error {
			args := c.Args()
			if len(args) < min {
				return c.Send(fmt.Sprintf(
					"❗ Недостаточно аргументов\nИспользование: %s",
					usage,
				))
			}
			return next(c)
		}
	}
}

// IgnoreBots ignores messages from bots.
func IgnoreBots() maxbot.MiddlewareFunc {
	return func(next maxbot.HandlerFunc) maxbot.HandlerFunc {
		return func(c maxbot.Context) error {
			if c.Sender().IsBot {
				return nil
			}
			return next(c)
		}
	}
}

// Chain combines multiple middleware into one.
func Chain(middleware ...maxbot.MiddlewareFunc) maxbot.MiddlewareFunc {
	return func(next maxbot.HandlerFunc) maxbot.HandlerFunc {
		for i := len(middleware) - 1; i >= 0; i-- {
			next = middleware[i](next)
		}
		return next
	}
}

// RateLimit limits requests per user per time window.
func RateLimit(max int, window time.Duration) maxbot.MiddlewareFunc {
	type userRequests struct {
		count     int
		resetTime time.Time
	}

	requests := make(map[int64]*userRequests)

	return func(next maxbot.HandlerFunc) maxbot.HandlerFunc {
		return func(c maxbot.Context) error {
			userID := c.Sender().ID
			now := time.Now()

			req, exists := requests[userID]
			if !exists || now.After(req.resetTime) {
				requests[userID] = &userRequests{
					count:     1,
					resetTime: now.Add(window),
				}
				return next(c)
			}

			if req.count >= max {
				remaining := req.resetTime.Sub(now)
				return c.Send(fmt.Sprintf(
					"⏱ Превышен лимит запросов. Попробуйте через %v",
					remaining.Round(time.Second),
				))
			}

			req.count++
			return next(c)
		}
	}
}

// FilterWords filters messages containing banned words.
func FilterWords(words []string, caseSensitive bool) maxbot.MiddlewareFunc {
	if !caseSensitive {
		for i, w := range words {
			words[i] = strings.ToLower(w)
		}
	}

	return func(next maxbot.HandlerFunc) maxbot.HandlerFunc {
		return func(c maxbot.Context) error {
			text := c.Text()
			if !caseSensitive {
				text = strings.ToLower(text)
			}

			for _, word := range words {
				if strings.Contains(text, word) {
					c.Delete()
					return c.Send("⚠️ Ваше сообщение содержит запрещенные слова")
				}
			}

			return next(c)
		}
	}
}

// Metrics collects basic metrics about bot usage.
type Metrics struct {
	TotalMessages   int64
	TotalCallbacks  int64
	UniqueUsers     map[int64]bool
	CommandsUsed    map[string]int64
	AverageResponse time.Duration
}

// Middleware returns a middleware function that collects metrics.
func (m *Metrics) Middleware() maxbot.MiddlewareFunc {
	if m.UniqueUsers == nil {
		m.UniqueUsers = make(map[int64]bool)
	}
	if m.CommandsUsed == nil {
		m.CommandsUsed = make(map[string]int64)
	}

	return func(next maxbot.HandlerFunc) maxbot.HandlerFunc {
		return func(c maxbot.Context) error {
			start := time.Now()

			m.UniqueUsers[c.Sender().ID] = true

			if c.Message() != nil {
				m.TotalMessages++

				text := c.Text()
				if len(text) > 0 && text[0] == '/' {
					cmd := strings.Fields(text)[0]
					m.CommandsUsed[cmd]++
				}
			}

			if c.Callback() != nil {
				m.TotalCallbacks++
			}

			err := next(c)

			duration := time.Since(start)
			m.AverageResponse = (m.AverageResponse + duration) / 2

			return err
		}
	}
}

// GetStats returns formatted statistics.
func (m *Metrics) GetStats() string {
	return fmt.Sprintf(`📊 Статистика бота:

💬 Всего сообщений: %d
🔘 Всего callback: %d
👥 Уникальных пользователей: %d
⚡ Средняя скорость ответа: %v

📈 Популярные команды:`,
		m.TotalMessages,
		m.TotalCallbacks,
		len(m.UniqueUsers),
		m.AverageResponse.Round(time.Millisecond),
	)
}
