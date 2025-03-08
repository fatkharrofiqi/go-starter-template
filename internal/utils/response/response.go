package response

import (
	"github.com/gin-gonic/gin"
)

// ErrorResponseData defines the structure of an error response
type ErrorResponseData struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Errors  interface{} `json:"errors,omitempty"`
}

// SuccessResponseData defines the structure of a success response
type SuccessResponseData struct {
	Status int         `json:"status"`
	Data   interface{} `json:"data,omitempty"`
}

// ErrorResponse handles both grouped validation errors (map), flat error messages (slice), and single error messages (string)
func ErrorResponse(ctx *gin.Context, status int, message string, errors ...interface{}) {
	resp := ErrorResponseData{
		Status:  status,
		Message: message,
	}

	// Handle optional errors
	if len(errors) > 0 && errors[0] != nil {
		switch e := errors[0].(type) {
		case error:
			// Convert Go error to string
			resp.Errors = e.Error()
		case []error:
			// Convert slice of errors to slice of strings
			var errStrings []string
			for _, err := range e {
				errStrings = append(errStrings, err.Error())
			}
			resp.Errors = errStrings
		case []string:
			// Directly assign if it's already a []string
			resp.Errors = e
		case map[string][]string:
			// Assign directly if it's a grouped error map
			resp.Errors = e
		default:
			// Assign any other type as is
			resp.Errors = e
		}
	}

	ctx.JSON(status, resp)
}

// SuccessResponse sends a successful JSON response
func SuccessResponse(ctx *gin.Context, status int, data interface{}) {
	ctx.JSON(status, SuccessResponseData{
		Status: status,
		Data:   data,
	})
}
