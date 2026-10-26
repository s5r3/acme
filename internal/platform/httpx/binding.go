package httpx

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"

	"example.com/acme/internal/shared/page"
)

// Validator adapts go-playground/validator to echo.Validator and converts
// failures into a Validation error keyed by the wire field name.
type Validator struct {
	v *validator.Validate
}

// NewValidator is registered as Echo's Validator by both binaries.
func NewValidator() *Validator {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(fieldName)
	return &Validator{v: v}
}

// Validate implements echo.Validator.
func (val *Validator) Validate(i any) error {
	err := val.v.Struct(i)
	if err == nil {
		return nil
	}
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return err
	}
	fields := make(map[string]string, len(verrs))
	for _, fe := range verrs {
		fields[fieldPath(fe)] = message(fe)
	}
	return Validation(fields)
}

// BindAndValidate binds the request into dst and validates it.
func BindAndValidate(c *echo.Context, dst any) error {
	if err := c.Bind(dst); err != nil {
		return ErrMalformedRequest.Wrap(err)
	}
	return c.Validate(dst)
}

// List is the one list envelope used by every collection endpoint.
type List[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
	Total    int64 `json:"total"`
}

// NewList maps a page result to the API envelope.
func NewList[E, T any](r page.Result[E], mapFn func(E) T) List[T] {
	items := make([]T, 0, len(r.Items))
	for _, e := range r.Items {
		items = append(items, mapFn(e))
	}
	return List[T]{Items: items, Page: r.Page, PageSize: r.PageSize, Total: r.Total}
}

func fieldName(f reflect.StructField) string {
	for _, tag := range []string{"json", "form", "query", "param"} {
		name := strings.SplitN(f.Tag.Get(tag), ",", 2)[0]
		if name == "-" {
			return ""
		}
		if name != "" {
			return name
		}
	}
	return f.Name
}

// fieldPath drops the root struct name: "CreateOrderRequest.items[0].sku" -> "items[0].sku".
func fieldPath(fe validator.FieldError) string {
	ns := fe.Namespace()
	if i := strings.IndexByte(ns, '.'); i >= 0 {
		return ns[i+1:]
	}
	return fe.Field()
}

func message(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		if fe.Kind() == reflect.String {
			return "must be at least " + fe.Param() + " characters"
		}
		return "must be at least " + fe.Param()
	case "max":
		if fe.Kind() == reflect.String {
			return "must be at most " + fe.Param() + " characters"
		}
		return "must be at most " + fe.Param()
	case "gt":
		return "must be greater than " + fe.Param()
	case "gte":
		return "must be " + fe.Param() + " or more"
	case "oneof":
		return "must be one of: " + fe.Param()
	case "uuid":
		return "must be a valid UUID"
	default:
		return "is invalid"
	}
}
