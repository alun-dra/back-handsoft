package handlers

type ErrorResponse struct {
	Message string `json:"message" example:"Unauthorized"`
}

type NoContentResponse struct{}
