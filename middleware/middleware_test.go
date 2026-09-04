package middleware

import (
	"testing"
	"time"

	"github.com/demen1n/maxbot"
)

// fakeContext is a minimal maxbot.Context implementation for unit-testing
// middleware in isolation, without a real Bot or network calls.
type fakeContext struct {
	sender   *maxbot.User
	chat     *maxbot.Chat
	text     string
	message  *maxbot.Message
	callback *maxbot.CallbackQuery
	store    map[string]interface{}

	sendErr error
	sent    []interface{}

	respondCalled bool
	respondErr    error

	deleteCalled bool
	deleteErr    error

	editErr error
	edited  []interface{}
}

var _ maxbot.Context = (*fakeContext)(nil)

func (c *fakeContext) Bot() *maxbot.Bot                { return nil }
func (c *fakeContext) Update() maxbot.Update           { return maxbot.Update{} }
func (c *fakeContext) Message() *maxbot.Message        { return c.message }
func (c *fakeContext) Callback() *maxbot.CallbackQuery { return c.callback }
func (c *fakeContext) Sender() *maxbot.User            { return c.sender }
func (c *fakeContext) Chat() *maxbot.Chat              { return c.chat }
func (c *fakeContext) Text() string                    { return c.text }
func (c *fakeContext) Args() []string                  { return nil }
func (c *fakeContext) Payload() string                 { return "" }

func (c *fakeContext) Send(what interface{}, opts ...interface{}) error {
	c.sent = append(c.sent, what)
	return c.sendErr
}
func (c *fakeContext) Reply(what interface{}, opts ...interface{}) error {
	return c.Send(what, opts...)
}
func (c *fakeContext) Edit(what interface{}, opts ...interface{}) error {
	c.edited = append(c.edited, what)
	return c.editErr
}
func (c *fakeContext) Delete() error {
	c.deleteCalled = true
	return c.deleteErr
}
func (c *fakeContext) Respond(opts ...*maxbot.CallbackResponse) error {
	c.respondCalled = true
	return c.respondErr
}
func (c *fakeContext) Get(key string) interface{} {
	if c.store == nil {
		return nil
	}
	return c.store[key]
}
func (c *fakeContext) Set(key string, val interface{}) {
	if c.store == nil {
		c.store = make(map[string]interface{})
	}
	c.store[key] = val
}

func okHandler() maxbot.HandlerFunc {
	return func(maxbot.Context) error { return nil }
}

func TestLogger(t *testing.T) {
	called := false
	next := func(maxbot.Context) error { called = true; return nil }
	h := Logger()(next)

	if err := h(&fakeContext{sender: &maxbot.User{Name: "A", Username: "a"}, text: "hi"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected next handler to run")
	}

	// Callback branch (no text).
	called = false
	if err := h(&fakeContext{
		sender:   &maxbot.User{Name: "A", Username: "a"},
		callback: &maxbot.CallbackQuery{Payload: "p1"},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected next handler to run for callback update")
	}
}

func TestAutoRespond(t *testing.T) {
	h := AutoRespond()(okHandler())

	c := &fakeContext{callback: &maxbot.CallbackQuery{CallbackID: "cb1"}}
	if err := h(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !c.respondCalled {
		t.Error("expected Respond to be called for a callback update")
	}

	c2 := &fakeContext{}
	if err := h(c2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c2.respondCalled {
		t.Error("expected Respond not to be called without a callback")
	}
}

func TestRecoverCatchesPanic(t *testing.T) {
	var caught error
	next := func(maxbot.Context) error { panic("boom") }
	h := Recover(func(err error) { caught = err })(next)

	c := &fakeContext{sender: &maxbot.User{}}
	if err := h(c); err != nil {
		t.Fatalf("Recover must swallow the panic and return nil, got %v", err)
	}
	if caught == nil {
		t.Error("expected onPanic callback to be invoked")
	}
	if len(c.sent) != 1 {
		t.Errorf("expected a user-facing error message to be sent, got %v", c.sent)
	}
}

func TestRecoverPassesThroughWithoutPanic(t *testing.T) {
	h := Recover()(okHandler())
	if err := h(&fakeContext{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWhitelist(t *testing.T) {
	h := Whitelist(1, 2)(okHandler())

	c := &fakeContext{sender: &maxbot.User{ID: 1}}
	if err := h(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.sent) != 0 {
		t.Error("allowed user should reach the handler without a rejection message")
	}

	c2 := &fakeContext{sender: &maxbot.User{ID: 99}}
	if err := h(c2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c2.sent) != 1 {
		t.Error("disallowed user should get a rejection message")
	}
}

func TestBlacklist(t *testing.T) {
	h := Blacklist(1)(okHandler())

	c := &fakeContext{sender: &maxbot.User{ID: 1}}
	if err := h(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.sent) != 0 {
		t.Error("blocked user should get no message, just be silently dropped")
	}

	called := false
	h2 := Blacklist(1)(func(maxbot.Context) error { called = true; return nil })
	if err := h2(&fakeContext{sender: &maxbot.User{ID: 2}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("non-blocked user should reach the handler")
	}
}

func TestThrottle(t *testing.T) {
	h := Throttle(time.Hour)(okHandler())
	c := &fakeContext{sender: &maxbot.User{ID: 1}}

	if err := h(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.sent) != 0 {
		t.Error("first call should pass through")
	}

	if err := h(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.sent) != 1 {
		t.Error("second call within the window should be throttled")
	}
}

func TestRestrictChatTypeAllowsMatching(t *testing.T) {
	h := RestrictChatType("group")(okHandler())
	c := &fakeContext{chat: &maxbot.Chat{Type: "group"}}
	if err := h(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.sent) != 0 {
		t.Error("matching chat type should pass through")
	}
}

func TestRestrictChatTypeRejectsOthers(t *testing.T) {
	h := RestrictChatType("group")(okHandler())

	c := &fakeContext{chat: &maxbot.Chat{Type: "dialog"}}
	if err := h(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.sent) != 1 {
		t.Error("mismatched chat type should be rejected")
	}

	c2 := &fakeContext{}
	if err := h(c2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c2.sent) != 1 {
		t.Error("nil chat should be rejected")
	}
}

func TestOnlyPrivateAndOnlyGroups(t *testing.T) {
	priv := OnlyPrivate()(okHandler())
	c := &fakeContext{chat: &maxbot.Chat{Type: "dialog"}}
	if err := priv(c); err != nil || len(c.sent) != 0 {
		t.Errorf("expected dialog chat to pass OnlyPrivate, sent=%v err=%v", c.sent, err)
	}

	groups := OnlyGroups()(okHandler())
	c2 := &fakeContext{chat: &maxbot.Chat{Type: "channel"}}
	if err := groups(c2); err != nil || len(c2.sent) != 0 {
		t.Errorf("expected channel chat to pass OnlyGroups, sent=%v err=%v", c2.sent, err)
	}

	c3 := &fakeContext{chat: &maxbot.Chat{Type: "dialog"}}
	if err := groups(c3); err != nil || len(c3.sent) != 1 {
		t.Errorf("expected dialog chat to be rejected by OnlyGroups, sent=%v err=%v", c3.sent, err)
	}
}

func TestCommandArgs(t *testing.T) {
	h := CommandArgs(2, "/cmd <a> <b>")(okHandler())

	c := &fakeContext{}
	c.text = "/cmd one"
	// Args() on fakeContext returns nil regardless of text, simulating "no args".
	if err := h(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.sent) != 1 {
		t.Error("expected usage message when args are missing")
	}
}

// argsContext extends fakeContext with a real Args() implementation, needed
// to exercise CommandArgs' passing branch.
type argsContext struct {
	fakeContext
	args []string
}

func (c *argsContext) Args() []string { return c.args }

func TestCommandArgsPassesWithEnoughArgs(t *testing.T) {
	h := CommandArgs(2, "/cmd <a> <b>")(okHandler())
	c := &argsContext{args: []string{"a", "b"}}
	if err := h(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.sent) != 0 {
		t.Error("expected handler to run without a usage message")
	}
}

func TestIgnoreBots(t *testing.T) {
	h := IgnoreBots()(okHandler())

	called := false
	h2 := IgnoreBots()(func(maxbot.Context) error { called = true; return nil })
	if err := h2(&fakeContext{sender: &maxbot.User{IsBot: false}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected human sender to reach the handler")
	}

	if err := h(&fakeContext{sender: &maxbot.User{IsBot: true}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChain(t *testing.T) {
	var order []string
	mw := func(name string) maxbot.MiddlewareFunc {
		return func(next maxbot.HandlerFunc) maxbot.HandlerFunc {
			return func(c maxbot.Context) error {
				order = append(order, name)
				return next(c)
			}
		}
	}

	h := Chain(mw("a"), mw("b"), mw("c"))(func(maxbot.Context) error {
		order = append(order, "handler")
		return nil
	})

	if err := h(&fakeContext{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"a", "b", "c", "handler"}
	if len(order) != len(want) {
		t.Fatalf("expected order %v, got %v", want, order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("expected order %v, got %v", want, order)
		}
	}
}

func TestRateLimit(t *testing.T) {
	h := RateLimit(2, time.Hour)(okHandler())
	c := &fakeContext{sender: &maxbot.User{ID: 1}}

	if err := h(c); err != nil || len(c.sent) != 0 {
		t.Fatalf("request 1 should pass, sent=%v err=%v", c.sent, err)
	}
	if err := h(c); err != nil || len(c.sent) != 0 {
		t.Fatalf("request 2 should pass, sent=%v err=%v", c.sent, err)
	}
	if err := h(c); err != nil || len(c.sent) != 1 {
		t.Fatalf("request 3 should be rate-limited, sent=%v err=%v", c.sent, err)
	}
}

func TestRateLimitResetsAfterWindow(t *testing.T) {
	h := RateLimit(1, 20*time.Millisecond)(okHandler())
	c := &fakeContext{sender: &maxbot.User{ID: 1}}

	if err := h(c); err != nil || len(c.sent) != 0 {
		t.Fatalf("first request should pass, sent=%v err=%v", c.sent, err)
	}
	time.Sleep(30 * time.Millisecond)
	if err := h(c); err != nil || len(c.sent) != 0 {
		t.Fatalf("request after window reset should pass, sent=%v err=%v", c.sent, err)
	}
}

func TestFilterWords(t *testing.T) {
	h := FilterWords([]string{"spam"}, false)(okHandler())

	c := &fakeContext{text: "this is SPAM content"}
	if err := h(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !c.deleteCalled {
		t.Error("expected message to be deleted")
	}
	if len(c.sent) != 1 {
		t.Error("expected a warning message to be sent")
	}

	called := false
	h2 := FilterWords([]string{"spam"}, false)(func(maxbot.Context) error { called = true; return nil })
	if err := h2(&fakeContext{text: "clean text"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected clean text to reach the handler")
	}
}

func TestFilterWordsCaseSensitive(t *testing.T) {
	h := FilterWords([]string{"Spam"}, true)(okHandler())

	called := false
	h2 := FilterWords([]string{"Spam"}, true)(func(maxbot.Context) error { called = true; return nil })
	if err := h2(&fakeContext{text: "spam lowercase does not match"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("case-sensitive filter should not match differing case")
	}

	c := &fakeContext{text: "contains Spam exactly"}
	if err := h(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !c.deleteCalled {
		t.Error("expected exact-case match to be filtered")
	}
}

func TestMetricsMiddleware(t *testing.T) {
	m := &Metrics{}
	h := m.Middleware()(okHandler())

	msg := &maxbot.Message{Body: &maxbot.MessageBody{Text: "/start"}}
	c := &fakeContext{sender: &maxbot.User{ID: 1}, message: msg, text: "/start"}
	if err := h(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cb := &fakeContext{sender: &maxbot.User{ID: 2}, callback: &maxbot.CallbackQuery{}}
	if err := h(cb); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.TotalMessages != 1 {
		t.Errorf("expected TotalMessages=1, got %d", m.TotalMessages)
	}
	if m.TotalCallbacks != 1 {
		t.Errorf("expected TotalCallbacks=1, got %d", m.TotalCallbacks)
	}
	if len(m.UniqueUsers) != 2 {
		t.Errorf("expected 2 unique users, got %d", len(m.UniqueUsers))
	}
	if m.CommandsUsed["/start"] != 1 {
		t.Errorf("expected /start counted once, got %d", m.CommandsUsed["/start"])
	}

	stats := m.GetStats()
	if stats == "" {
		t.Error("expected non-empty stats string")
	}
}

func TestMetricsGetStatsZeroValue(t *testing.T) {
	m := &Metrics{}
	if got := m.GetStats(); got == "" {
		t.Error("expected non-empty stats for a fresh Metrics")
	}
}
