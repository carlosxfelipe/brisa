package validation

import (
	"fmt"
	"net/mail"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/carlosxfelipe/brisa/core"
)

var (
	uuidPattern       = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	compiledMetaCache sync.Map // map[reflect.Type]*compiledStructMeta
)

type compiledFieldMeta struct {
	index      int
	name       string
	fieldType  reflect.Type
	rules      []string
	customMsgs map[string]string
}

type compiledEmbeddedMeta struct {
	index        int
	embeddedType reflect.Type
}

type compiledStructMeta struct {
	fields   []compiledFieldMeta
	embedded []compiledEmbeddedMeta
}

func getCompiledStructMeta(t reflect.Type) *compiledStructMeta {
	if cached, ok := compiledMetaCache.Load(t); ok {
		return cached.(*compiledStructMeta)
	}
	meta := &compiledStructMeta{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if ft, ok := core.EmbeddedStructType(field); ok {
			meta.embedded = append(meta.embedded, compiledEmbeddedMeta{
				index:        i,
				embeddedType: ft,
			})
			continue
		}
		if !field.IsExported() {
			continue
		}
		validateTag := field.Tag.Get("validate")
		msgTag := field.Tag.Get("msg")
		if msgTag == "" {
			msgTag = field.Tag.Get("message")
		}
		var rules []string
		if validateTag != "" {
			for _, r := range splitRules(validateTag) {
				if r = strings.TrimSpace(r); r != "" {
					rules = append(rules, r)
				}
			}
		}
		meta.fields = append(meta.fields, compiledFieldMeta{
			index:      i,
			name:       fieldIdentifier(field),
			fieldType:  field.Type,
			rules:      rules,
			customMsgs: parseMessageTag(msgTag),
		})
	}
	compiledMetaCache.Store(t, meta)
	return meta
}

// Validator is an interface that structs can implement for custom validation logic.
type Validator interface {
	Validate() error
}

// ContextValidator is an interface that structs can implement for validation with Context access.
type ContextValidator interface {
	Validate(ctx *core.Context) error
}

// ValidationError holds structured field validation errors.
type ValidationError struct {
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors"`
	Status  int                 `json:"status"`
}

func (v *ValidationError) Error() string {
	var parts []string
	for field, msgs := range v.Errors {
		parts = append(parts, fmt.Sprintf("%s: %s", field, strings.Join(msgs, ", ")))
	}
	return fmt.Sprintf("validation failed: %s", strings.Join(parts, "; "))
}

// HTTPStatus returns the status code used when this error is rendered (422 by default).
func (v *ValidationError) HTTPStatus() int {
	if v.Status == 0 {
		return 422
	}
	return v.Status
}

// FieldErrors returns the map of field-level validation error messages.
func (v *ValidationError) FieldErrors() map[string][]string {
	return v.Errors
}

func isStdLibStruct(t reflect.Type) bool {
	if t == reflect.TypeOf(time.Time{}) {
		return true
	}
	pkg := t.PkgPath()
	return strings.HasPrefix(pkg, "mime/") || strings.HasPrefix(pkg, "net/") || strings.HasPrefix(pkg, "io")
}

func isKnownRule(s string) bool {
	if s == "required" || s == "email" || s == "url" || s == "uuid" || s == "nowarn" {
		return true
	}
	parts := strings.SplitN(s, "=", 2)
	switch parts[0] {
	case "min", "max", "len", "oneof":
		return true
	default:
		return false
	}
}

func splitRules(tag string) []string {
	var rules []string
	for _, part := range strings.Split(tag, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if len(rules) > 0 && strings.HasPrefix(rules[len(rules)-1], "oneof=") && !isKnownRule(part) {
			rules[len(rules)-1] += "," + part
		} else {
			rules = append(rules, part)
		}
	}
	return rules
}

func parseMessageTag(tag string) map[string]string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil
	}
	msgs := make(map[string]string)
	for _, part := range strings.Split(tag, "|") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if idx := strings.Index(part, ":"); idx != -1 {
			rule := strings.TrimSpace(part[:idx])
			msg := strings.TrimSpace(part[idx+1:])
			if rule != "" && msg != "" {
				msgs[rule] = msg
			}
		} else {
			msgs["_"] = part
		}
	}
	return msgs
}

func resolveCustomMessage(customMsgs map[string]string, ruleName, fullName, defaultMsg string) string {
	if len(customMsgs) == 0 {
		return defaultMsg
	}
	msg, ok := customMsgs[ruleName]
	if !ok {
		msg, ok = customMsgs["_"]
	}
	if !ok {
		return defaultMsg
	}
	if strings.Contains(msg, "{field}") {
		return strings.ReplaceAll(msg, "{field}", fullName)
	}
	return msg
}

// SilenceTagWarnings controls whether startup warning logs are printed for suspicious validation tags.
var SilenceTagWarnings bool

// TagWarningLogger, if set, redirects tag warning messages.
var TagWarningLogger func(warning string)

// CheckTags verifies that every `validate` tag on the struct type t only uses
// known rules with valid parameters, recursing into nested structs and slices.
func CheckTags(t reflect.Type) error {
	return checkTagsInternal(t, make(map[reflect.Type]bool))
}

func checkTagsInternal(t reflect.Type, seen map[reflect.Type]bool) error {
	if t == nil {
		return nil
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || isStdLibStruct(t) || seen[t] {
		return nil
	}
	seen[t] = true
	defer delete(seen, t)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("validate")
		name := fieldIdentifier(field)

		msgTag := field.Tag.Get("msg")
		if msgTag == "" {
			msgTag = field.Tag.Get("message")
		}
		if msgTag != "" {
			if tag == "" {
				return fmt.Errorf("field %q: tag `msg` defined without a `validate` tag", name)
			}
			declaredRules := make(map[string]bool)
			for _, r := range splitRules(tag) {
				parts := strings.SplitN(strings.TrimSpace(r), "=", 2)
				declaredRules[parts[0]] = true
			}
			customMsgs := parseMessageTag(msgTag)
			for ruleKey := range customMsgs {
				if ruleKey == "_" {
					continue
				}
				if !isKnownRule(ruleKey) {
					return fmt.Errorf("field %q: unknown rule %q in `msg` tag", name, ruleKey)
				}
				if !declaredRules[ruleKey] {
					return fmt.Errorf("field %q: `msg` specifies message for rule %q, but rule is not present in `validate:%q`", name, ruleKey, tag)
				}
			}
		}

		if tag != "" {
			for _, rule := range splitRules(tag) {
				rule = strings.TrimSpace(rule)
				if rule == "" {
					continue
				}
				parts := strings.SplitN(rule, "=", 2)
				switch parts[0] {
				case "required", "email", "url", "uuid", "nowarn", "omitempty":
				case "min", "max":
					if len(parts) < 2 || parts[1] == "" {
						return fmt.Errorf("field %q: rule %q requires a numeric parameter (e.g. %s=3)", name, parts[0], parts[0])
					}
					if _, err := strconv.ParseFloat(parts[1], 64); err != nil {
						return fmt.Errorf("field %q: rule %q has an invalid numeric parameter %q", name, parts[0], parts[1])
					}
				case "len":
					if len(parts) < 2 || parts[1] == "" {
						return fmt.Errorf("field %q: rule %q requires an integer parameter (e.g. len=5)", name, parts[0])
					}
					if _, err := strconv.Atoi(parts[1]); err != nil {
						return fmt.Errorf("field %q: rule %q has an invalid integer parameter %q", name, parts[0], parts[1])
					}
				case "oneof":
					if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
						return fmt.Errorf("field %q: rule %q requires at least one choice (e.g. oneof=a,b,c)", name, parts[0])
					}
				default:
					return fmt.Errorf("field %q: unknown validation rule %q", name, parts[0])
				}
			}
		}

		ft := unwrapStructType(field.Type)
		if ft.Kind() == reflect.Struct && !isStdLibStruct(ft) {
			if err := checkTagsInternal(ft, seen); err != nil {
				return err
			}
		}
	}
	getCompiledStructMeta(t)
	return nil
}

// CheckTagWarnings inspects struct type t for suspicious validation tags.
func CheckTagWarnings(t reflect.Type) []string {
	var warnings []string
	checkTagWarningsInternal(t, make(map[reflect.Type]bool), nil, &warnings)
	return warnings
}

func checkTagWarningsInternal(t reflect.Type, seen map[reflect.Type]bool, shadowed map[string]bool, warnings *[]string) {
	if t == nil {
		return
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || isStdLibStruct(t) || seen[t] {
		return
	}
	seen[t] = true
	defer delete(seen, t)

	mergedShadowed := collectShadowedFields(t, shadowed)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if _, ok := core.EmbeddedStructType(field); ok {
			continue
		}
		if !field.IsExported() {
			continue
		}
		name := fieldIdentifier(field)
		if shadowed[name] {
			continue
		}

		suppressed := field.Tag.Get("brisa") == "nowarn"
		tag := field.Tag.Get("validate")
		hasRequired := false
		if tag != "" {
			for _, rule := range splitRules(tag) {
				rule = strings.TrimSpace(rule)
				if rule == "nowarn" {
					suppressed = true
				}
				if rule == "required" {
					hasRequired = true
				}
			}
		}

		typeName := t.Name()
		if typeName == "" {
			typeName = "struct"
		}

		if hasRequired && !suppressed && isPrimitiveRejectingZero(field.Type) {
			kindStr := field.Type.String()
			zeroVal := zeroValueString(field.Type)
			msg := fmt.Sprintf(
				"field %q in %s is a non-pointer %s with validate:\"required\"; in Go, zero value (%s) will fail validation. Use *%s if zero values are allowed, or silence with brisa.SilenceTagWarnings(true).",
				name, typeName, kindStr, zeroVal, kindStr,
			)
			*warnings = append(*warnings, msg)
		}

		ft := unwrapStructType(field.Type)
		if ft.Kind() == reflect.Struct && !isStdLibStruct(ft) {
			checkTagWarningsInternal(ft, seen, nil, warnings)
		}
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		ft, ok := core.EmbeddedStructType(field)
		if !ok {
			continue
		}
		checkTagWarningsInternal(ft, seen, mergedShadowed, warnings)
	}
}

func isPrimitiveRejectingZero(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func zeroValueString(t reflect.Type) string {
	if t.Kind() == reflect.Bool {
		return "false"
	}
	return "0"
}

// ValidateStruct validates a struct instance using both its validate tags and Validator interface.
func ValidateStruct(ctx *core.Context, target any) *ValidationError {
	val := reflect.ValueOf(target)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	errMap := make(map[string][]string)

	if val.Kind() == reflect.Struct {
		validateStructRecursive(ctx, val, "", errMap, nil, make(map[reflect.Type]bool))
	}

	if v, ok := target.(Validator); ok {
		if err := v.Validate(); err != nil {
			errMap["_general"] = append(errMap["_general"], err.Error())
		}
	}

	if cv, ok := target.(ContextValidator); ok {
		if err := cv.Validate(ctx); err != nil {
			errMap["_general"] = append(errMap["_general"], err.Error())
		}
	}

	if len(errMap) > 0 {
		return &ValidationError{
			Message: "validation failed",
			Errors:  errMap,
			Status:  422,
		}
	}

	return nil
}

func fieldIdentifier(field reflect.StructField) string {
	if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
		parts := strings.Split(jsonTag, ",")
		if parts[0] != "" {
			return parts[0]
		}
	}
	return field.Name
}

func validateStructRecursive(ctx *core.Context, val reflect.Value, prefix string, errMap map[string][]string, shadowed map[string]bool, embedding map[reflect.Type]bool) {
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct || isStdLibStruct(val.Type()) {
		return
	}

	t := val.Type()
	if embedding[t] {
		return
	}
	embedding[t] = true
	defer delete(embedding, t)

	mergedShadowed := collectShadowedFields(t, shadowed)
	meta := getCompiledStructMeta(t)

	for _, fm := range meta.fields {
		if shadowed[fm.name] {
			continue
		}
		fullName := joinPrefix(prefix, fm.name)
		fieldVal := val.Field(fm.index)

		for _, rule := range fm.rules {
			if err := applyValidationRule(fullName, fieldVal, rule); err != "" {
				ruleName := strings.SplitN(rule, "=", 2)[0]
				err = resolveCustomMessage(fm.customMsgs, ruleName, fullName, err)
				errMap[fullName] = append(errMap[fullName], err)
			}
		}

		ft := fm.fieldType
		if ft.Kind() == reflect.Pointer {
			if !fieldVal.IsNil() {
				validateStructRecursive(ctx, fieldVal.Elem(), fullName, errMap, nil, embedding)
			}
		} else if ft.Kind() == reflect.Struct && !isStdLibStruct(ft) {
			validateStructRecursive(ctx, fieldVal, fullName, errMap, nil, embedding)
		} else if ft.Kind() == reflect.Slice {
			elemType := ft.Elem()
			if elemType.Kind() == reflect.Pointer {
				elemType = elemType.Elem()
			}
			if elemType.Kind() == reflect.Struct && !isStdLibStruct(elemType) {
				for idx := 0; idx < fieldVal.Len(); idx++ {
					item := fieldVal.Index(idx)
					itemPrefix := fmt.Sprintf("%s[%d]", fullName, idx)
					validateStructRecursive(ctx, item, itemPrefix, errMap, nil, embedding)
				}
			}
		}
	}

	for _, em := range meta.embedded {
		fieldVal := val.Field(em.index)
		if fieldVal.Kind() == reflect.Pointer {
			if fieldVal.IsNil() {
				validateNilEmbedded(em.embeddedType, prefix, errMap, mergedShadowed)
				continue
			}
			validateStructRecursive(ctx, fieldVal.Elem(), prefix, errMap, mergedShadowed, embedding)
			continue
		}
		validateStructRecursive(ctx, fieldVal, prefix, errMap, mergedShadowed, embedding)
	}

	if val.CanAddr() {
		if v, ok := val.Addr().Interface().(Validator); ok && prefix != "" {
			if err := v.Validate(); err != nil {
				errMap[prefix] = append(errMap[prefix], err.Error())
			}
		}
		if cv, ok := val.Addr().Interface().(ContextValidator); ok && prefix != "" {
			if err := cv.Validate(ctx); err != nil {
				errMap[prefix] = append(errMap[prefix], err.Error())
			}
		}
	}
}

func validateNilEmbedded(ft reflect.Type, prefix string, errMap map[string][]string, shadowed map[string]bool) {
	for i := 0; i < ft.NumField(); i++ {
		f := ft.Field(i)
		if subType, ok := core.EmbeddedStructType(f); ok {
			validateNilEmbedded(subType, prefix, errMap, shadowed)
			continue
		}
		if !f.IsExported() {
			continue
		}
		name := fieldIdentifier(f)
		if shadowed[name] {
			continue
		}
		fullName := joinPrefix(prefix, name)
		tag := f.Tag.Get("validate")
		msgTag := f.Tag.Get("msg")
		if msgTag == "" {
			msgTag = f.Tag.Get("message")
		}
		customMsgs := parseMessageTag(msgTag)
		if tag != "" {
			for _, rule := range splitRules(tag) {
				if strings.TrimSpace(rule) == "required" {
					msg := fmt.Sprintf("field %s is required", fullName)
					msg = resolveCustomMessage(customMsgs, "required", fullName, msg)
					errMap[fullName] = append(errMap[fullName], msg)
				}
			}
		}
	}
}

func applyValidationRule(field string, val reflect.Value, rule string) string {
	parts := strings.SplitN(rule, "=", 2)
	ruleName := parts[0]
	ruleParam := ""
	if len(parts) > 1 {
		ruleParam = parts[1]
	}

	if ruleName != "required" {
		for val.Kind() == reflect.Pointer || val.Kind() == reflect.Interface {
			if val.IsNil() {
				return ""
			}
			val = val.Elem()
		}
	}

	switch ruleName {
	case "nowarn", "omitempty":
		return ""
	case "required":
		if isZero(val) {
			return fmt.Sprintf("field %s is required", field)
		}

	case "min":
		if ruleParam == "" {
			return ""
		}
		minVal, err := strconv.ParseFloat(ruleParam, 64)
		if err != nil {
			return ""
		}
		if num, unit, ok := toComparableValue(val); ok && num < minVal {
			if unit == " items" {
				return fmt.Sprintf("field %s must contain at least %v items", field, minVal)
			}
			return fmt.Sprintf("field %s must be at least %v%s", field, minVal, unit)
		}

	case "max":
		if ruleParam == "" {
			return ""
		}
		maxVal, err := strconv.ParseFloat(ruleParam, 64)
		if err != nil {
			return ""
		}
		if num, unit, ok := toComparableValue(val); ok && num > maxVal {
			if unit == " items" {
				return fmt.Sprintf("field %s must not contain more than %v items", field, maxVal)
			}
			return fmt.Sprintf("field %s must not exceed %v%s", field, maxVal, unit)
		}

	case "len":
		if ruleParam == "" {
			return ""
		}
		targetLen, err := strconv.Atoi(ruleParam)
		if err != nil {
			return ""
		}
		switch val.Kind() {
		case reflect.String:
			if utf8.RuneCountInString(val.String()) != targetLen {
				return fmt.Sprintf("field %s must be exactly %d characters", field, targetLen)
			}
		case reflect.Slice, reflect.Array, reflect.Map:
			if val.Len() != targetLen {
				return fmt.Sprintf("field %s must contain exactly %d items", field, targetLen)
			}
		}

	case "oneof":
		if ruleParam == "" {
			return ""
		}
		if val.Kind() == reflect.String && val.String() == "" {
			return ""
		}
		var validOptions []string
		if strings.Contains(ruleParam, ",") {
			validOptions = strings.Split(ruleParam, ",")
		} else {
			validOptions = strings.Fields(ruleParam)
		}
		for i := range validOptions {
			validOptions[i] = strings.TrimSpace(validOptions[i])
		}

		actualVal := fmt.Sprintf("%v", val.Interface())
		found := false
		for _, opt := range validOptions {
			if opt == actualVal {
				found = true
				break
			}
		}
		if !found {
			return fmt.Sprintf("field %s must be one of [%s]", field, strings.Join(validOptions, ", "))
		}

	case "email":
		if val.Kind() == reflect.String {
			s := val.String()
			if s != "" {
				_, err := mail.ParseAddress(s)
				if err != nil || !strings.Contains(s, "@") || !strings.Contains(s, ".") {
					return fmt.Sprintf("field %s must be a valid email address", field)
				}
			}
		}

	case "url":
		if val.Kind() == reflect.String {
			s := val.String()
			if s != "" {
				u, err := url.ParseRequestURI(s)
				if err != nil || u.Scheme == "" || u.Host == "" {
					return fmt.Sprintf("field %s must be a valid URL", field)
				}
			}
		}

	case "uuid":
		if val.Kind() == reflect.String {
			s := val.String()
			if s != "" {
				if !uuidPattern.MatchString(s) {
					return fmt.Sprintf("field %s must be a valid UUID", field)
				}
			}
		}
	}

	return ""
}

func isZero(val reflect.Value) bool {
	if !val.IsValid() {
		return true
	}
	switch val.Kind() {
	case reflect.String:
		return strings.TrimSpace(val.String()) == ""
	case reflect.Slice, reflect.Map:
		return val.IsNil() || val.Len() == 0
	case reflect.Pointer, reflect.Interface:
		return val.IsNil()
	default:
		return val.IsZero()
	}
}

func unwrapStructType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() == reflect.Slice {
		t = t.Elem()
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
	}
	return t
}

func collectShadowedFields(t reflect.Type, shadowed map[string]bool) map[string]bool {
	merged := make(map[string]bool, len(shadowed)+t.NumField())
	for k, v := range shadowed {
		merged[k] = v
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if _, ok := core.EmbeddedStructType(field); ok {
			continue
		}
		merged[fieldIdentifier(field)] = true
	}
	return merged
}

func joinPrefix(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

func toComparableValue(val reflect.Value) (float64, string, bool) {
	switch val.Kind() {
	case reflect.String:
		return float64(utf8.RuneCountInString(val.String())), " characters", true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(val.Uint()), "", true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(val.Int()), "", true
	case reflect.Float32, reflect.Float64:
		return val.Float(), "", true
	case reflect.Slice:
		return float64(val.Len()), " items", true
	default:
		return 0, "", false
	}
}
