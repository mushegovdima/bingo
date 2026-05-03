package notifications

// Notification carries a codename for a single push.
// Template data is derived automatically by reflecting over the struct fields — see templateservice.Render.
type Notification interface {
	Codename() string
	Args() map[string]string
}

type NotificationBase struct {
	User NotificationUser
}

// TemplateVar describes a single {{placeholder}} accepted by a template body.
type TemplateVar struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// NotificationUser holds recipient profile data substituted into {{User.*}} placeholders
// during template rendering. Pass nil for broadcast notifications with no specific user.
type NotificationUser struct {
	Name     string `label:"Имя"`
	Username string `label:"Username"`
}

// All is the canonical list of every notification type, as zero-value instances.
// Use it wherever you need to iterate over all known notifications
// (e.g. building the codename→notification map, seeding DB templates, etc.).
var All = []Notification{
	ClaimSubmitted{},
	ClaimCompleted{},
	ClaimCancelled{},
	TaskApproved{},
	TaskRejected{},
	SeasonAvailable{},
}

// --- reward notifications ---

// ClaimSubmitted is sent to a user after they successfully submit a reward claim.
type ClaimSubmitted struct {
	NotificationBase
	RewardTitle string `label:"Название приза"`
	SpentCoins  int    `label:"Потрачено монет"`
}

func (n ClaimSubmitted) Codename() string        { return "claim_submitted" }
func (n ClaimSubmitted) Args() map[string]string { return ArgsOf(n) }

// ClaimCompleted is sent when an admin marks a reward claim as completed.
type ClaimCompleted struct {
	NotificationBase
	RewardTitle string `label:"Название приза"`
}

func (n ClaimCompleted) Codename() string        { return "claim_completed" }
func (n ClaimCompleted) Args() map[string]string { return ArgsOf(n) }

// ClaimCancelled is sent when a reward claim is cancelled and coins are refunded.
type ClaimCancelled struct {
	NotificationBase
	RewardTitle   string `label:"Название приза"`
	RefundedCoins int    `label:"Возвращено монет"`
}

func (n ClaimCancelled) Codename() string        { return "claim_cancelled" }
func (n ClaimCancelled) Args() map[string]string { return ArgsOf(n) }

// --- submission notifications ---

// TaskApproved is sent when an admin approves a task submission.
type TaskApproved struct {
	NotificationBase
	TaskTitle string `label:"Название задачи"`
	Coins     int    `label:"Монеты"`
}

func (n TaskApproved) Codename() string        { return "task_approved" }
func (n TaskApproved) Args() map[string]string { return ArgsOf(n) }

// TaskRejected is sent when a manager rejects a task submission.
type TaskRejected struct {
	NotificationBase
	TaskTitle     string `label:"Название задачи"`
	ReviewComment string `label:"Комментарий проверяющего"`
}

func (n TaskRejected) Codename() string        { return "task_rejected" }
func (n TaskRejected) Args() map[string]string { return ArgsOf(n) }

// --- season notifications ---

// SeasonAvailable is broadcast to all active Telegram users when a season is published.
type SeasonAvailable struct {
	NotificationBase
	Title     string `label:"Название сезона"`
	StartDate string `label:"Дата начала"`
	EndDate   string `label:"Дата окончания"`
}

func (n SeasonAvailable) Codename() string        { return "season_available" }
func (n SeasonAvailable) Args() map[string]string { return ArgsOf(n) }
