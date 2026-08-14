package realtime

import (
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) IssueTicket(c *fiber.Ctx) error {
	userID, ok := c.Locals(authctx.MetadataUserID).(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": fiber.StatusUnauthorized, "msg": "missing user"})
	}
	user := authctx.User{ID: userID}
	if username, ok := c.Locals(authctx.MetadataUsername).(string); ok {
		user.Username = username
	}
	ticket, expiresIn, err := h.tickets.Issue(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": fiber.StatusInternalServerError, "msg": err.Error()})
	}
	return c.JSON(fiber.Map{"code": 20000, "data": fiber.Map{"ticket": ticket, "expiresIn": expiresIn}})
}
