package notifications

import (
	"testing"
)

// --- Substitute ---

func TestSubstitute_ReplacesKnownPlaceholders(t *testing.T) {
	body := "Привет, {{User.Name}}! Твой приз: {{RewardTitle}}."
	args := map[string]string{
		"User.Name":   "Алиса",
		"RewardTitle": "iPhone 15",
	}
	got := Substitute(body, args)
	want := "Привет, Алиса! Твой приз: iPhone 15."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSubstitute_UnknownPlaceholderLeftAsIs(t *testing.T) {
	body := "Потрачено {{SpentCoins}} монет, {{unknown}} не трогаем."
	args := map[string]string{"SpentCoins": "50"}
	got := Substitute(body, args)
	want := "Потрачено 50 монет, {{unknown}} не трогаем."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSubstitute_EmptyArgsReturnsBodyUnchanged(t *testing.T) {
	body := "Сезон {{Title}} начался!"
	got := Substitute(body, map[string]string{})
	if got != body {
		t.Errorf("got %q, want %q", got, body)
	}
}

func TestSubstitute_EmptyBodyReturnsEmpty(t *testing.T) {
	got := Substitute("", map[string]string{"Key": "val"})
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestSubstitute_NoPlaceholdersInBody(t *testing.T) {
	body := "Простое сообщение без плейсхолдеров."
	got := Substitute(body, map[string]string{"RewardTitle": "iPhone"})
	if got != body {
		t.Errorf("got %q, want %q", got, body)
	}
}

// --- UserArgs ---

func TestUserArgs_KeysHaveUserPrefix(t *testing.T) {
	u := NotificationUser{Name: "Боб", Username: "bob42"}
	args := u.UserArgs()

	if v, ok := args["User.Name"]; !ok || v != "Боб" {
		t.Errorf("User.Name: got %q (ok=%v), want %q", v, ok, "Боб")
	}
	if v, ok := args["User.Username"]; !ok || v != "bob42" {
		t.Errorf("User.Username: got %q (ok=%v), want %q", v, ok, "bob42")
	}
}

func TestUserArgs_ZeroValueUser(t *testing.T) {
	u := NotificationUser{}
	args := u.UserArgs()
	if v := args["User.Name"]; v != "" {
		t.Errorf("User.Name: got %q, want empty", v)
	}
	if v := args["User.Username"]; v != "" {
		t.Errorf("User.Username: got %q, want empty", v)
	}
}

func TestUserArgs_NoExtraKeys(t *testing.T) {
	u := NotificationUser{Name: "x", Username: "y"}
	args := u.UserArgs()
	// NotificationUser has exactly 2 fields
	if len(args) != 2 {
		t.Errorf("got %d keys, want 2: %v", len(args), args)
	}
}

// --- ArgsOf ---

func TestArgsOf_ClaimSubmitted_ExtractsAllFields(t *testing.T) {
	n := ClaimSubmitted{RewardTitle: "MacBook", SpentCoins: 200}
	args := n.Args()

	cases := map[string]string{
		"RewardTitle":   "MacBook",
		"SpentCoins":    "200",
		"User.Name":     "",
		"User.Username": "",
	}
	for key, want := range cases {
		if got := args[key]; got != want {
			t.Errorf("args[%q] = %q, want %q", key, got, want)
		}
	}
}

func TestArgsOf_ClaimCancelled_ExtractsRefundedCoins(t *testing.T) {
	n := ClaimCancelled{RewardTitle: "Кофемашина", RefundedCoins: 75}
	args := n.Args()

	if got := args["RefundedCoins"]; got != "75" {
		t.Errorf("RefundedCoins: got %q, want %q", got, "75")
	}
	if got := args["RewardTitle"]; got != "Кофемашина" {
		t.Errorf("RewardTitle: got %q, want %q", got, "Кофемашина")
	}
}

func TestArgsOf_SeasonAvailable_ExtractsDateFields(t *testing.T) {
	n := SeasonAvailable{Title: "Зима 2026", StartDate: "2026-12-01", EndDate: "2026-12-31"}
	args := n.Args()

	if got := args["Title"]; got != "Зима 2026" {
		t.Errorf("Title: got %q, want %q", got, "Зима 2026")
	}
	if got := args["StartDate"]; got != "2026-12-01" {
		t.Errorf("StartDate: got %q, want %q", got, "2026-12-01")
	}
	if got := args["EndDate"]; got != "2026-12-31" {
		t.Errorf("EndDate: got %q, want %q", got, "2026-12-31")
	}
}

func TestArgsOf_FieldsWithoutLabelTagAreExcluded(t *testing.T) {
	// NotificationBase embeds NotificationUser but the base struct itself has no label tag.
	// Only leaf fields tagged with `label:` must appear.
	args := ArgsOf(TaskApproved{TaskTitle: "Пробежка", Coins: 10})
	for k := range args {
		if k == "User" || k == "NotificationBase" {
			t.Errorf("unexpected key %q: struct fields without label tag must not appear", k)
		}
	}
}

// --- End-to-end: ArgsOf + UserArgs → Substitute ---

func TestSubstitute_EndToEnd_WorkerPath(t *testing.T) {
	// Имитирует путь worker.sendBatch:
	// 1. args из job (notification-specific) + user fields → merge → Substitute.
	n := ClaimSubmitted{RewardTitle: "AirPods", SpentCoins: 50}
	u := NotificationUser{Name: "Максим", Username: "max"}

	body := "{{User.Name}}, твой заказ на {{RewardTitle}} принят. Списано: {{SpentCoins}} монет."

	notifArgs := n.Args()
	userArgs := u.UserArgs()
	merged := make(map[string]string, len(notifArgs)+len(userArgs))
	for k, v := range notifArgs {
		merged[k] = v
	}
	for k, v := range userArgs {
		merged[k] = v
	}

	got := Substitute(body, merged)
	want := "Максим, твой заказ на AirPods принят. Списано: 50 монет."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSubstitute_EndToEnd_UserArgsOverrideNotifArgs(t *testing.T) {
	// Проверяем, что при конфликте ключей overlay (userArgs) побеждает base.
	base := map[string]string{"User.Name": "из_нотификации", "Key": "val"}
	overlay := map[string]string{"User.Name": "из_пользователя"}

	merged := make(map[string]string, len(base)+len(overlay))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range overlay {
		merged[k] = v
	}

	got := Substitute("{{User.Name}} {{Key}}", merged)
	want := "из_пользователя val"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
