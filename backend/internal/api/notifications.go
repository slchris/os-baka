package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/os-baka/backend/internal/model"
)

type NotificationHandler struct{}

func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{}
}

func (h *NotificationHandler) List(c *gin.Context) {
	var notifications []model.Notification
	model.DB.Order("created_at desc").Find(&notifications)

	type NotificationView struct {
		ID        uint   `json:"id"`
		Title     string `json:"title"`
		Message   string `json:"message"`
		Type      string `json:"type"`
		Timestamp string `json:"timestamp"` // simplified relative time handled by frontend
		Read      bool   `json:"read"`
	}

	var response []NotificationView
	for _, n := range notifications {
		response = append(response, NotificationView{
			ID:        n.ID,
			Title:     n.Title,
			Message:   n.Message,
			Type:      n.Type,
			Timestamp: n.CreatedAt.Format(time.RFC3339),
			Read:      n.Read,
		})
	}

	// If empty, return empty list
	if response == nil {
		response = []NotificationView{}
	}

	c.JSON(http.StatusOK, response)
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}

	var notification model.Notification
	if result := model.DB.First(&notification, id); result.Error != nil {
		ErrorResponse(c, http.StatusNotFound, "Notification not found")
		return
	}

	notification.Read = true
	model.DB.Save(&notification)

	c.JSON(http.StatusOK, gin.H{"success": true})
}
