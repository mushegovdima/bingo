package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gorilla/sessions"
	"go.mod/internal/api/response"
	"go.mod/internal/config"
	"go.mod/internal/domain"
)

const sessionName = "sid"
const ctxKeySessionID = "session_id"
const ctxKeyUserID = "user_id"

type sessionContextKey struct{}

// userGetter извлекает пользователя по ID (реализуется UserService — данные из кэша).
type userGetter interface {
	GetById(ctx context.Context, id int64) (*domain.User, error)
}

// SessionCtx — данные сессии, доступные в контексте каждого запроса.
type SessionCtx struct {
	SessionID int64
	UserID    int64
}

// NewStore создаёт gorilla CookieStore с HMAC-подписью (и опциональным шифрованием).
// hashKey — обязателен, encryptKey — передайте nil если шифрование не нужно.
func NewStore(env string, cfg config.Config) *sessions.CookieStore {
	store := sessions.NewCookieStore([]byte(cfg.SessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   cfg.SessionTTLMinutes * 60,
		HttpOnly: true,
		Secure:   true,
		SameSite: func() http.SameSite {
			if env == "prod" {
				return http.SameSiteLaxMode
			}
			return http.SameSiteNoneMode
		}(),
	}
	return store
}

// SessionFromContext возвращает SessionCtx из контекста, или nil если сессия отсутствует.
func SessionFromContext(ctx context.Context) *SessionCtx {
	s, _ := ctx.Value(sessionContextKey{}).(*SessionCtx)
	return s
}

// WithSession возвращает новый контекст с установленной сессией.
// Используется в тестах для инъекции аутентифицированной сессии.
func WithSession(ctx context.Context, sess *SessionCtx) context.Context {
	return context.WithValue(ctx, sessionContextKey{}, sess)
}

// SessionRefresh читает горилла-сессию, проверяет подпись и кладёт данные в context.
// Скользящий срок обеспечивается тем, что gorilla пересохраняет куку при каждом Save().
func SessionRefresh(store *sessions.CookieStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := slog.Default().With(slog.String("path", r.URL.Path), slog.String("method", r.Method))

			session, err := store.Get(r, sessionName)
			if err != nil {
				// Подпись невалидна — удаляем испорченную куку
				log.Debug("session: invalid cookie signature", slog.Any("err", err))
				session.Options.MaxAge = -1
				session.Save(r, w)
				session.Options.MaxAge = store.Options.MaxAge
				next.ServeHTTP(w, r)
				return
			}
			if session.IsNew {
				// Куки нет вообще — просто продолжаем без сессии
				log.Debug("session: no cookie")
				next.ServeHTTP(w, r)
				return
			}

			sessionID, okS := session.Values[ctxKeySessionID].(int64)
			userID, okU := session.Values[ctxKeyUserID].(int64)
			if !okS || !okU {
				log.Debug("session: cookie present but values missing", slog.Any("keys", session.Values))
				next.ServeHTTP(w, r)
				return
			}
			log.Debug("session: ok", slog.Int64("user_id", userID), slog.Int64("session_id", sessionID))

			// Перевыпускаем куку (скользящий срок)
			session.Save(r, w)

			ctx := context.WithValue(r.Context(), sessionContextKey{}, &SessionCtx{
				SessionID: sessionID,
				UserID:    userID,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// SaveSession записывает session_id и user_id в подписанную куку.
func SaveSession(w http.ResponseWriter, r *http.Request, store *sessions.CookieStore, sessionID, userID int64) error {
	session, err := store.Get(r, sessionName)
	if err != nil {
		return err
	}
	session.Values[ctxKeySessionID] = sessionID
	session.Values[ctxKeyUserID] = userID
	return session.Save(r, w)
}

// RequireAuth — middleware для защищённых роутов.
// Возвращает 401 если сессия отсутствует или невалидна.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			slog.Default().Warn("unauthorized",
				slog.String("path", r.URL.Path),
				slog.String("method", r.Method),
				slog.Any("cookie", r.Header),
			)
			response.Err(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ClearSession удаляет куку сессии.
func ClearSession(w http.ResponseWriter, r *http.Request, store *sessions.CookieStore) error {
	session, err := store.Get(r, sessionName)
	if err != nil {
		return err
	}
	session.Options.MaxAge = -1
	return session.Save(r, w)
}

// RequireRole возвращает middleware, которое проверяет роли пользователя из кэша.
// Заодно проверяет что пользователь не заблокирован и не удалён.
// Должен быть после RequireAuth.
func RequireRole(users userGetter, roles ...domain.UserRole) func(http.Handler) http.Handler {
	allowed := make(map[domain.UserRole]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess := SessionFromContext(r.Context())
			if sess == nil {
				response.Err(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			user, err := users.GetById(r.Context(), sess.UserID)
			if err != nil || user == nil || user.IsBlocked {
				response.Err(w, http.StatusForbidden, "forbidden")
				return
			}

			for _, role := range user.Roles {
				if _, ok := allowed[role]; ok {
					next.ServeHTTP(w, r)
					return
				}
			}

			response.Err(w, http.StatusForbidden, "forbidden")
		})
	}
}
