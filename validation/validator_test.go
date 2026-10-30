package validation_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/carlosxfelipe/brisa/validation"
)

type SampleInput struct {
	Name  string   `json:"name" validate:"required,min=3,max=20" msg:"required:Name is mandatory"`
	Email string   `json:"email" validate:"required,email"`
	URL   string   `json:"url" validate:"url"`
	UUID  string   `json:"uuid" validate:"uuid"`
	Code  string   `json:"code" validate:"len=4"`
	Role  string   `json:"role" validate:"oneof=admin,editor,viewer"`
	Tags  []string `json:"tags" validate:"min=1,max=3"`
}

func (s *SampleInput) Validate() error {
	if s.Name == "forbidden" {
		return errors.New("forbidden name")
	}
	return nil
}

func TestValidateStructRules(t *testing.T) {
	if err := validation.CheckTags(reflect.TypeOf(SampleInput{})); err != nil {
		t.Fatalf("unexpected tag check error: %v", err)
	}

	valid := SampleInput{
		Name:  "Carlos",
		Email: "carlos@example.com",
		URL:   "https://example.com",
		UUID:  "550e8400-e29b-41d4-a716-446655440000",
		Code:  "ABCD",
		Role:  "admin",
		Tags:  []string{"go", "desktop"},
	}
	if valErr := validation.ValidateStruct(nil, &valid); valErr != nil {
		t.Fatalf("expected valid struct, got: %v", valErr)
	}

	invalid := SampleInput{
		Name:  "",
		Email: "not-an-email",
		URL:   "invalid-url",
		UUID:  "123",
		Code:  "AB",
		Role:  "guest",
		Tags:  nil,
	}
	valErr := validation.ValidateStruct(nil, &invalid)
	if valErr == nil {
		t.Fatal("expected validation error, got nil")
	}
	if valErr.Errors["name"][0] != "Name is mandatory" {
		t.Fatalf("expected custom required message, got %v", valErr.Errors["name"])
	}
	for _, field := range []string{"email", "url", "uuid", "code", "role", "tags"} {
		if len(valErr.Errors[field]) == 0 {
			t.Fatalf("expected validation error on field %q", field)
		}
	}

	customInvalid := valid
	customInvalid.Name = "forbidden"
	valErr2 := validation.ValidateStruct(nil, &customInvalid)
	if valErr2 == nil || len(valErr2.Errors["_general"]) == 0 {
		t.Fatalf("expected custom Validator error, got %+v", valErr2)
	}
}
