package main

import (
	"context"
	"flag"
	"fmt"
	"go.mod/internal/config"
	"go.mod/internal/db"
	dbmodels "go.mod/internal/db"
	"go.mod/internal/db/repository"
	userdomain "go.mod/internal/domain/user"
	"log/slog"
	"math/rand"
	"os"
	"time"
)

var firstNames = []string{
	"Алекс", "Иван", "Мария", "Дарья", "Сергей",
	"Анна", "Дмитрий", "Елена", "Максим", "Ольга",
	"Андрей", "Светлана", "Никита", "Юлия", "Артём",
	"Татьяна", "Кирилл", "Наталья", "Михаил", "Виктория",
	"Павел", "Ксения", "Роман", "Алина", "Денис",
	"Евгения", "Вадим", "Полина", "Антон", "Кристина",
}

var lastNames = []string{
	"Иванов", "Смирнова", "Кузнецов", "Попова", "Васильев",
	"Петрова", "Соколов", "Михайлова", "Новиков", "Фёдорова",
	"Морозов", "Волкова", "Алексеев", "Лебедева", "Семёнов",
}

func main() {
	env := flag.String("env", "dev", "environment (dev|prod)")
	configPath := flag.String("config", "", "path to config file (default: {env}.env)")
	count := flag.Int("count", 30, "number of test users to create")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.LoadConfig(*env, *configPath)
	if err != nil {
		logger.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	pg, err := db.NewDB(cfg, logger)
	if err != nil {
		logger.Error("failed to connect to postgres", slog.Any("error", err))
		os.Exit(1)
	}
	defer pg.Close()

	ctx := context.Background()
	repo := repository.NewUserRepository(ctx, pg.DB(), logger)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	created := 0
	for i := 0; i < *count; i++ {
		firstName := firstNames[i%len(firstNames)]
		lastName := lastNames[rng.Intn(len(lastNames))]
		name := fmt.Sprintf("%s %s", firstName, lastName)
		username := fmt.Sprintf("test_user_%d", i+1)
		telegramID := int64(9_000_000_000) + rng.Int63n(999_999_999)

		user := &dbmodels.User{
			TelegramID: telegramID,
			Name:       name,
			Username:   username,
			PhotoURL:   "",
			Roles:      []userdomain.UserRole{userdomain.Resident},
		}

		if err := repo.Insert(ctx, user); err != nil {
			logger.Info("user already exists, skipping",
				slog.String("username", username),
			)
			continue
		}
		created++
		logger.Info("user created",
			slog.Int64("id", user.ID),
			slog.String("name", name),
			slog.String("username", username),
		)
	}

	fmt.Printf("\nDone. Created %d/%d test users.\n", created, *count)
}
