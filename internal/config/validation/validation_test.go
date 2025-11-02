package validation

import (
    "net/http"
    "strings"
    "testing"

    "github.com/gofiber/fiber/v2"
    "github.com/stretchr/testify/require"

    "go-starter-template/internal/utils/errcode"
)

// Test struct with validation tags
type testReq struct {
    Name     string `json:"name" validate:"required,alpha,min=3,max=20"`
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}

// Test struct without json tag to verify fallback to lowercase field name
type noJsonTag struct {
    NoTag string `validate:"required"`
}

// Test struct to exercise default tag branch using an unsupported tag in switch
type urlReq struct {
    Website string `json:"website" validate:"url"`
}

func TestValidate(t *testing.T) {
    v := NewValidation()

    cases := []struct {
        name       string
        input      interface{}
        assertFunc func(t *testing.T, err error)
    }{
        {
            name: "Success",
            input: &testReq{Name: "Alice", Email: "alice@example.com", Password: "secret123"},
            assertFunc: func(t *testing.T, err error) { require.NoError(t, err) },
        },
        {
            name: "MultipleErrors_Messages",
            input: &testReq{Name: "Al1", Email: "", Password: "123"},
            assertFunc: func(t *testing.T, err error) {
                require.Error(t, err)
                vErr, ok := err.(*ValidationError)
                require.True(t, ok)
                require.Equal(t, "Validation failed", vErr.Message)
                require.Contains(t, vErr.Errors, "name")
                require.Contains(t, vErr.Errors, "email")
                require.Contains(t, vErr.Errors, "password")
                require.Contains(t, vErr.Errors["name"], "name must contain only alphabetic characters")
                require.Contains(t, vErr.Errors["email"], "email is required")
                require.Contains(t, vErr.Errors["password"], "password must be at least 8 characters long")
            },
        },
        {
            name: "MaxMessage",
            input: &testReq{Name: strings.Repeat("A", 21), Email: "alice@example.com", Password: "secret123"},
            assertFunc: func(t *testing.T, err error) {
                require.Error(t, err)
                vErr, ok := err.(*ValidationError)
                require.True(t, ok)
                require.Contains(t, vErr.Errors, "name")
                require.Contains(t, vErr.Errors["name"], "name must not exceed 20 characters")
            },
        },
        {
            name: "JsonTagFallback",
            input: &noJsonTag{},
            assertFunc: func(t *testing.T, err error) {
                require.Error(t, err)
                vErr, ok := err.(*ValidationError)
                require.True(t, ok)
                require.Contains(t, vErr.Errors, "notag")
                require.Contains(t, vErr.Errors["notag"], "notag is required")
            },
        },
        {
            name: "DefaultTagMessage",
            input: &urlReq{Website: "not_a_url"},
            assertFunc: func(t *testing.T, err error) {
                require.Error(t, err)
                vErr, ok := err.(*ValidationError)
                require.True(t, ok)
                require.Contains(t, vErr.Errors, "website")
                require.Contains(t, vErr.Errors["website"], "website is invalid (url)")
            },
        },
        {
            name: "UnexpectedValidationError",
            input: &[]string{"x"},
            assertFunc: func(t *testing.T, err error) {
                require.Error(t, err)
                _, isValErr := err.(*ValidationError)
                require.False(t, isValErr)
                require.Contains(t, err.Error(), "unexpected validation error")
            },
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            err := v.Validate(tc.input)
            tc.assertFunc(t, err)
        })
    }
}

func TestValidate_Success(t *testing.T) {
    v := NewValidation()
    req := &testReq{
        Name:     "Alice",
        Email:    "alice@example.com",
        Password: "secret123",
    }
    err := v.Validate(req)
    require.NoError(t, err)
}

func TestValidate_ErrorsAndMessages(t *testing.T) {
    v := NewValidation()
    // Violates multiple rules: missing email, short password, name not alpha
    req := &testReq{
        Name:     "Al1", // not alpha
        Email:    "",    // required
        Password: "123", // min 8
    }
    err := v.Validate(req)
    require.Error(t, err)

    vErr, ok := err.(*ValidationError)
    require.True(t, ok, "expected ValidationError")
    require.Equal(t, "Validation failed", vErr.Message)

    // Expect aggregated messages per json field
    require.Contains(t, vErr.Errors, "name")
    require.Contains(t, vErr.Errors, "email")
    require.Contains(t, vErr.Errors, "password")

    // Specific message checks (format defined in implementation)
    require.Contains(t, vErr.Errors["name"], "name must contain only alphabetic characters")
    require.Contains(t, vErr.Errors["email"], "email is required")
    require.Contains(t, vErr.Errors["password"], "password must be at least 8 characters long")
}

func TestValidate_MaxMessage(t *testing.T) {
    v := NewValidation()
    // Name exceeds max length 20
    req := &testReq{
        Name:     strings.Repeat("A", 21),
        Email:    "alice@example.com",
        Password: "secret123",
    }
    err := v.Validate(req)
    require.Error(t, err)
    vErr, ok := err.(*ValidationError)
    require.True(t, ok)
    require.Contains(t, vErr.Errors, "name")
    require.Contains(t, vErr.Errors["name"], "name must not exceed 20 characters")
}

func TestValidate_DefaultTagMessage(t *testing.T) {
    v := NewValidation()
    req := &urlReq{Website: "not_a_url"}
    err := v.Validate(req)
    require.Error(t, err)
    vErr, ok := err.(*ValidationError)
    require.True(t, ok)
    require.Contains(t, vErr.Errors, "website")
    require.Contains(t, vErr.Errors["website"], "website is invalid (url)")
}

func TestValidate_UnexpectedValidationError(t *testing.T) {
    v := NewValidation()
    // Passing a non-struct should produce validator.InvalidValidationError
    data := []string{"x"}
    err := v.Validate(&data)
    require.Error(t, err)
    // Ensure this is not a ValidationError type and contains our message
    _, isValErr := err.(*ValidationError)
    require.False(t, isValErr)
    require.Contains(t, err.Error(), "unexpected validation error")
}

func TestValidate_JsonTagFallback(t *testing.T) {
    v := NewValidation()
    req := &noJsonTag{}
    err := v.Validate(req)
    require.Error(t, err)
    vErr, ok := err.(*ValidationError)
    require.True(t, ok)
    // Fallback should be lowercase field name "notag"
    require.Contains(t, vErr.Errors, "notag")
    require.Contains(t, vErr.Errors["notag"], "notag is required")
}

func TestParseAndValidate_BadRequestOnBodyParse(t *testing.T) {
    app := fiber.New()
    v := NewValidation()

    app.Post("/", func(c *fiber.Ctx) error {
        var req testReq
        err := v.ParseAndValidate(c, &req)
        if err != nil {
            // Map known errors to status codes via errcode
            if code, ok := errcode.GetHTTPStatus(err); ok {
                return c.Status(code).JSON(fiber.Map{"error": err.Error()})
            }
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
        }
        return c.SendStatus(fiber.StatusOK)
    })

    // Invalid JSON type for name (number into string) should trigger BodyParser error
    req := httptestNewJSONRequest("/", `{"name":1}`)
    resp, err := app.Test(req, -1)
    require.NoError(t, err)
    require.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// removed: superseded by table-driven ParseAndValidate test

// helper to build an HTTP request with JSON body for fiber.Test
func httptestNewJSONRequest(path, body string) *http.Request {
    req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    return req
}

// removed: superseded by table-driven ParseAndValidate test

func TestParseAndValidate(t *testing.T) {
    newApp := func() *fiber.App {
        app := fiber.New()
        v := NewValidation()
        app.Post("/", func(c *fiber.Ctx) error {
            var req testReq
            err := v.ParseAndValidate(c, &req)
            if err != nil {
                if code, ok := errcode.GetHTTPStatus(err); ok {
                    return c.Status(code).JSON(fiber.Map{"error": err.Error()})
                }
                return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
            }
            return c.SendStatus(fiber.StatusOK)
        })
        return app
    }

    cases := []struct {
        name         string
        body         string
        expectStatus int
    }{
        {name: "BadRequest_BodyParse", body: `{"name":1}`, expectStatus: fiber.StatusBadRequest},
        {name: "Success", body: `{"name":"Alice","email":"alice@example.com","password":"secret123"}`, expectStatus: fiber.StatusOK},
        {name: "ValidationError", body: `{"name":"Al1","email":"invalid","password":"123"}`, expectStatus: fiber.StatusBadRequest},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            app := newApp()
            req := httptestNewJSONRequest("/", tc.body)
            resp, err := app.Test(req, -1)
            require.NoError(t, err)
            require.Equal(t, tc.expectStatus, resp.StatusCode)
        })
    }
}