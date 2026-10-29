package notify

import (
	"context"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/handlers"
)

func (m *Module) Routes(r chi.Router) {
	r.Get("/notifications", handlers.GetAll(m.store.ListNotifications))
	r.Post("/notifications/read-all", handlers.GetAll(m.readAll))
	r.Delete("/notifications/{id}", handlers.Delete(m.store.DeleteNotification))
	r.Post("/notifications/{id}/read", handlers.Delete(m.store.MarkNotificationRead))
}

func (m *Module) readAll(ctx context.Context, userID string) (struct{}, error) {
	return struct{}{}, m.store.MarkAllNotificationsRead(ctx, userID)
}
