package core

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
)

// Empty represents an empty request or response payload in Brisa REPR commands.
type Empty struct{}

// Map is a shorthand alias for map[string]any.
type Map map[string]any

// RPCError represents a structured error returned by a Brisa command handler.
type RPCError struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Status  int                 `json:"status"`
	Details any                 `json:"details,omitempty"`
	Errors  map[string][]string `json:"errors,omitempty"`
}

func (e *RPCError) Error() string {
	return e.Message
}

// NewError creates a new structured RPCError with a status code and message.
func NewError(status int, code, message string) *RPCError {
	if status == 0 {
		status = 400
	}
	if code == "" {
		code = "BRISA_ERROR"
	}
	return &RPCError{
		Code:    code,
		Message: message,
		Status:  status,
	}
}

// ErrorResponse is the standard error wire format returned by the Brisa IPC bridge.
type ErrorResponse struct {
	Code      string              `json:"code"`
	Message   string              `json:"message"`
	Status    int                 `json:"status"`
	Command   string              `json:"command,omitempty"`
	RequestID string              `json:"requestId,omitempty"`
	Errors    map[string][]string `json:"errors,omitempty"`
	Details   any                 `json:"details,omitempty"`
}

// CommandDoc holds descriptive metadata for a registered IPC command.
type CommandDoc struct {
	Name        string
	Group       string
	Summary     string
	Description string
	Tags        []string
	Deprecated  bool
	RequestType reflect.Type
	ReplyType   reflect.Type
}

// FieldSchema describes a single struct field in the IPC schema and TypeScript generator.
type FieldSchema struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	TSType      string                 `json:"tsType"`
	Description string                 `json:"description,omitempty"`
	Example     any                    `json:"example,omitempty"`
	Required    bool                   `json:"required"`
	Rules       []string               `json:"rules,omitempty"`
	Properties  map[string]FieldSchema `json:"properties,omitempty"`
	Items       *FieldSchema           `json:"items,omitempty"`
}

// CommandSchema represents the full introspective schema for a single IPC command.
type CommandSchema struct {
	Name        string       `json:"name"`
	Group       string       `json:"group,omitempty"`
	Summary     string       `json:"summary,omitempty"`
	Description string       `json:"description,omitempty"`
	Tags        []string     `json:"tags,omitempty"`
	Deprecated  bool         `json:"deprecated,omitempty"`
	Input       *FieldSchema `json:"input,omitempty"`
	Output      *FieldSchema `json:"output,omitempty"`
}

// InspectorSpec describes the complete application IPC contract exposed at /brisa/schema.json.
type InspectorSpec struct {
	AppID       string          `json:"appId"`
	Title       string          `json:"title"`
	Version     string          `json:"version"`
	Description string          `json:"description,omitempty"`
	Commands    []CommandSchema `json:"commands"`
}

// EmbeddedStructType reports whether a struct field is an anonymous embedded struct
// that should have its fields promoted during binding, validation, and schema generation.
func EmbeddedStructType(field reflect.StructField) (reflect.Type, bool) {
	if !field.Anonymous {
		return nil, false
	}
	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		name, _, _ := strings.Cut(jsonTag, ",")
		if name != "" {
			return nil, false
		}
	}
	t := field.Type
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || t == reflect.TypeOf(time.Time{}) {
		return nil, false
	}
	return t, true
}

// BuildTypeSchema inspects a Go reflect.Type and produces a FieldSchema for documentation and TS SDK generation.
func BuildTypeSchema(t reflect.Type) *FieldSchema {
	if t == nil {
		return nil
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == reflect.TypeOf(Empty{}) || t == reflect.TypeOf(struct{}{}) {
		return nil
	}
	schema := buildFieldSchemaRecursive(t, make(map[reflect.Type]bool))
	return &schema
}

func buildFieldSchemaRecursive(t reflect.Type, seen map[reflect.Type]bool) FieldSchema {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == reflect.TypeOf(time.Time{}) {
		return FieldSchema{Type: "string (date-time)", TSType: "string"}
	}

	switch t.Kind() {
	case reflect.String:
		return FieldSchema{Type: "string", TSType: "string"}
	case reflect.Bool:
		return FieldSchema{Type: "boolean", TSType: "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return FieldSchema{Type: "number", TSType: "number"}
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return FieldSchema{Type: "string (base64)", TSType: "string"}
		}
		itemSchema := buildFieldSchemaRecursive(t.Elem(), seen)
		return FieldSchema{
			Type:   "array",
			TSType: itemSchema.TSType + "[]",
			Items:  &itemSchema,
		}
	case reflect.Map:
		valSchema := buildFieldSchemaRecursive(t.Elem(), seen)
		return FieldSchema{
			Type:   "object",
			TSType: fmt.Sprintf("Record<string, %s>", valSchema.TSType),
		}
	case reflect.Struct:
		if seen[t] {
			return FieldSchema{Type: "object", TSType: "Record<string, any>"}
		}
		seen[t] = true
		defer delete(seen, t)

		props := make(map[string]FieldSchema)
		collectStructFields(t, props, seen)

		var tsFields []string
		keys := make([]string, 0, len(props))
		for k := range props {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			prop := props[k]
			opt := "?"
			if prop.Required {
				opt = ""
			}
			tsFields = append(tsFields, fmt.Sprintf("%s%s: %s", k, opt, prop.TSType))
		}
		tsObj := "{ " + strings.Join(tsFields, "; ") + " }"
		if len(tsFields) == 0 {
			tsObj = "Record<string, never>"
		}
		return FieldSchema{
			Name:       t.Name(),
			Type:       "object",
			TSType:     tsObj,
			Properties: props,
		}
	default:
		return FieldSchema{Type: "any", TSType: "any"}
	}
}

func collectStructFields(t reflect.Type, props map[string]FieldSchema, seen map[reflect.Type]bool) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if embType, ok := EmbeddedStructType(f); ok {
			collectStructFields(embType, props, seen)
			continue
		}
		if !f.IsExported() {
			continue
		}
		jsonTag := f.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}
		fieldName := f.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				fieldName = parts[0]
			}
		}

		fs := buildFieldSchemaRecursive(f.Type, seen)
		fs.Name = fieldName
		fs.Description = f.Tag.Get("doc")
		if ex := f.Tag.Get("example"); ex != "" {
			fs.Example = ex
		}
		if vTag := f.Tag.Get("validate"); vTag != "" {
			for _, r := range strings.Split(vTag, ",") {
				r = strings.TrimSpace(r)
				if r == "" || r == "nowarn" {
					continue
				}
				if r == "required" {
					fs.Required = true
				}
				fs.Rules = append(fs.Rules, r)
			}
		}
		props[fieldName] = fs
	}
}
