package server

import (
	"github.com/gofiber/fiber/v2"
)

type errorResponse struct {
	Error string `json:"error"`
}

type deleteResponse struct {
	ID      int64  `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type updateResponse struct {
	ID      int64  `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type createResponse struct {
	Status string `json:"status"`
	Date   string `json:"date"`
	Type   string `json:"type"`
}

// writeJSON serializes value as JSON with a 200 status.
func writeJSON(c *fiber.Ctx, value any) error {
	return c.Status(fiber.StatusOK).JSON(value)
}

// writeCreated serializes value as JSON with a 201 status.
func writeCreated(c *fiber.Ctx, value any) error {
	return c.Status(fiber.StatusCreated).JSON(value)
}

// writeError responds with a JSON error payload using the given status code
// and message. The response body is always `{"error": "message"}`.
func writeError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(errorResponse{Error: message})
}
