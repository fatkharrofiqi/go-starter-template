package dto

type CsrfRequest struct {
	Path string `json:"path" validate:"max=100"`
}
