package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func BindAndValidate[T any](r *http.Request, target *T) error {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		return fmt.Errorf("malformed JSON response: %w", err)
	}
	defer r.Body.Close()

	if err := validate.Struct(target); err != nil {
		return err
	}

	return nil
}

func ParseRequest[T any](req *http.Request, payload *T) error {
	if err := BindAndValidate(req, payload); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			out := make([]string, len(ve))
			for _, fe := range ve {
				out = append(out, fmt.Sprintf("%s: failed validation on rule: %s.", fe.Field(), fe.Tag()))
			}
			return errors.New(strings.Join(out, "\n"))
		}

		return err
	}

	return nil
}
