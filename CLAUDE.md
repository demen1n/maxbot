# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run all tests
go test ./...

# Run a single test
go test -run TestName ./...

# Run tests with verbose output
go test -v ./...

# Build the example bot
go build ./cmd

# Format code
go fmt ./...

# Vet code
go vet ./...

# Tidy dependencies
go mod tidy
```

## Architecture

**Maxbot** is a Go library for building bots for the MAX messaging platform (Mail.ru messenger), inspired by [telebot](https://github.com/tucnak/telebot). Module path: `github.com/demen1n/maxbot`.

### Update flow

```
Poller (LongPoller or Webhook)
  → getUpdates() / HTTP handler
  → Update struct (buffered chan, size 64)
  → Bot.ProcessUpdate()
  → Bot.match() — priority: Callback → Command → Media type → Text → OnMessage
  → HandlerFunc (wrapped with middleware, LIFO order)
  → nativeContext methods
```

### Key types

- **`Bot`** (`bot.go`) — core struct; `Handle()` registers handlers; `Send()`, `Edit()`, `Delete()` are the main message operations. `Start()` blocks until `Stop()` is called.
- **`Context`** (`context.go`) — interface passed to every handler; provides `Send()`, `Reply()`, `Edit()`, `Delete()`, `Respond()`, `Sender()`, `Chat()`, `Text()`, `Args()`, etc.
    - `Send()` sends to the current **chat** (not the user's DM); falls back to the sender only when no chat is available.
    - `Reply()` sends a quoted reply using the incoming message's `mid`.
    - `Chat()` resolves the chat from both message updates and callback updates (via `CallbackQuery.Message`).
- **`Update` / `Message`** (`types.go`) — domain models; `Message` has custom JSON unmarshaling to auto-populate `ReplyTo` from `body.link`.
- **`CallbackQuery`** (`types.go`) — includes a `Message` field with the originating message, used for chat resolution in callback handlers.
- **`Poller`** (`poller.go`) — interface with two implementations: `LongPoller` (marker-based) and `Webhook` (HTTP server with optional secret verification; handler is non-blocking via buffered channel + goroutine fallback).
- **`Sendable`** / **`Editable`** / **`Recipient`** — interfaces for flexible argument passing to `Send()` / `Edit()`.
- **`ReplyMarkup`** / **`InlineButton`** (`markup.go`) — inline keyboard builder; `Data()` creates callback buttons, `URL()` creates link buttons.

### Handler registration

Special handler endpoint keys use the `\a` (bell) prefix to avoid collisions with user-defined strings:

| Key | Matches |
|-----|---------|
| `"\amessage"` | Any update |
| `"\atext"` | Plain text messages |
| `"\acallback"` | Callback queries (catch-all fallback) |
| `"\aphoto"` / `"\avideo"` / `"\aaudio"` / `"\adocument"` | Media by type |
| `/command` | Commands |
| `payload` | Exact callback payload match (checked before `OnCallback`) |

`Handle(*InlineButton)` registers the handler under `btn.Payload` (what the API sends back), not `btn.Data`. `btn.Data` is only a routing hint when `Payload` is empty.

### Recipient routing

`newSendMessage(to Recipient)` type-switches the recipient:
- `*Chat` → sets `chat_id` query param
- anything else (e.g. `*User`) → sets `user_id` query param

### Middleware

`middleware/middleware.go` contains 14+ middleware functions. They wrap handlers via function composition; when multiple are passed to `Handle()`, they are applied LIFO (last arg = outermost wrapper).

Notable: `Throttle`, `RateLimit`, and `Metrics` hold in-memory state that is lost on restart.

### API client

`api.go` communicates with `https://platform-api.max.ru`. The token is stored as-is and passed directly in the `Authorization` header (the MAX API does not use a `Bearer` prefix). `Raw()` is the generic request method. File uploads go through a two-step `GetUploadURL()` → `UploadFile()` flow.

Edit/delete operations use `?message_id=` query params (not path segments). `*Message` always goes through `editMessageByMid` / `deleteMessage` using the string `mid`; `StoredMessage` uses its integer ID.

### The `old/` directory

Contains pre-refactor code tagged `//go:build ignore`. It is excluded from compilation and ignored by git — do not edit or rely on it.
