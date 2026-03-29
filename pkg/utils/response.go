package utils

import "github.com/gofiber/fiber/v2"

type APIResponse struct {
	Message      string      `json:"message"`
	ResponseCode string      `json:"responseCode"`
	Data         interface{} `json:"data,omitempty"`
}

func SendResponse(c *fiber.Ctx, httpStatus int, responseCode string, message string, data interface{}) error {
	return c.Status(httpStatus).JSON(APIResponse{
		Message:      message,
		ResponseCode: responseCode,
		Data:         data,
	})
}