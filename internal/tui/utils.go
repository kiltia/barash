package tui

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"orb/runner/pkg/config"

	"go.uber.org/zap/zapcore"
)

func BuildNavigationForStruct(structValue any) []ConfigItem {
	return buildNavigationForValue(reflect.ValueOf(structValue))
}

func buildNavigationForValue(v reflect.Value) []ConfigItem {
	var items []ConfigItem
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if fieldType.Tag.Get("display") == "-" {
			continue
		}

		items = append(items, ConfigItem{
			Name:     fieldType.Name,
			Value:    field.Interface(),
			IsStruct: field.Kind() == reflect.Struct,
		})
	}

	return items
}

func GetValueByPath(cfg *config.Config, path []string) any {
	current := reflect.ValueOf(cfg).Elem()
	for _, segment := range path {
		current = current.FieldByName(segment)
	}
	return current.Interface()
}

// Field operations
func UpdateField(m FieldEditorModel) error {
	configValue := reflect.ValueOf(m.Config).Elem()

	for _, segment := range m.Path {
		configValue = configValue.FieldByName(segment)
	}

	field := configValue.FieldByName(m.EditField.Name)
	if !field.CanSet() {
		return fmt.Errorf("field cannot be set")
	}

	return setFieldValue(field, m.textInput.Value())
}

func setFieldValue(field reflect.Value, value string) error {
	switch field.Type() {
	case reflect.TypeOf(time.Duration(0)):
		duration, _ := time.ParseDuration(value)
		field.Set(reflect.ValueOf(duration))
	case reflect.TypeOf(zapcore.Level(0)):
		var level zapcore.Level
		err := level.UnmarshalText([]byte(value))
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(level))
	default:
		parsedValue := parseValue(value, field.Kind())
		field.Set(parsedValue.Convert(field.Type()))
	}
	return nil
}

func parseValue(value string, kind reflect.Kind) reflect.Value {
	switch kind {
	case reflect.Int, reflect.Int64:
		intVal, _ := strconv.ParseInt(value, 10, 64)
		return reflect.ValueOf(intVal)
	case reflect.Bool:
		boolVal, _ := strconv.ParseBool(value)
		return reflect.ValueOf(boolVal)
	default:
		return reflect.ValueOf(value)
	}
}

// Value formatting
func FormatValue(value any) string {
	switch v := value.(type) {
	case time.Duration:
		return v.String()
	case zapcore.Level:
		return v.String()
	case string:
		if v == "" {
			return `""`
		}
		return v
	case int, int64:
		return fmt.Sprintf("%d", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Config file operations
func LoadConfig(cfg *config.Config, path string) error {
	err := os.Setenv("CONFIG_FILE", path)
	if err != nil {
		return err
	}
	config.Init()
	config.Load(cfg)
	return nil
}

func SaveConfig(cfg *config.Config, path string) error {
	content := ConfigToEnv(cfg)
	return os.WriteFile(path, []byte(content), 0o644)
}

// Environment variable generation
func ConfigToEnv(cfg *config.Config) string {
	var result strings.Builder
	structToEnv(reflect.ValueOf(cfg).Elem(), "", &result)
	return result.String()
}

func structToEnv(v reflect.Value, prefix string, result *strings.Builder) {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if !field.CanInterface() {
			continue
		}

		envTag := fieldType.Tag.Get("env")
		if envTag == "" {
			continue
		}

		if field.Kind() == reflect.Struct {
			// For nested structs, extract prefix from tag like "env:\", prefix=API_\""
			newPrefix := extractEnvconfigPrefix(envTag)
			// Remove trailing underscore if present
			newPrefix = strings.TrimSuffix(newPrefix, "_")

			fullPrefix := newPrefix
			if prefix != "" {
				fullPrefix = prefix + "_" + newPrefix
			}
			structToEnv(field, fullPrefix, result)
		} else {
			// For simple fields, extract the env var name like "env:\"NAME\""
			envName := extractEnvconfigName(envTag)
			if envName != "" {
				fullEnvName := envName
				if prefix != "" {
					fullEnvName = prefix + "_" + envName
				}
				value := formatValueForEnv(field)
				if value != "" {
					fmt.Fprintf(result, "%s=%s\n", fullEnvName, value)
				}
			}
		}
	}
}

func extractEnvconfigPrefix(tag string) string {
	// Parse tags like "env:\", prefix=API_\"" or "env:\", prefix=CLICKHOUSE_\""
	parts := strings.SplitSeq(tag, ",")
	for part := range parts {
		part = strings.TrimSpace(part)
		if after, ok := strings.CutPrefix(part, "prefix="); ok {
			prefix := after
			return prefix
		}
	}
	return ""
}

func extractEnvconfigName(tag string) string {
	// Parse tags like "env:\"NAME\"" or "env:\"HOST, default=127.0.0.1\""
	if tag == "" {
		return ""
	}

	// Split by comma to handle tags with options
	parts := strings.Split(tag, ",")
	if len(parts) == 0 {
		return ""
	}

	// First part should be the env var name
	envName := strings.TrimSpace(parts[0])

	// Skip if it's a prefix definition or empty
	if envName == "" || strings.Contains(envName, "prefix=") {
		return ""
	}

	return envName
}

func formatValueForEnv(field reflect.Value) string {
	switch field.Kind() {
	case reflect.String:
		return field.String()
	case reflect.Int, reflect.Int64:
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			return field.Interface().(time.Duration).String()
		}
		return strconv.FormatInt(field.Int(), 10)
	case reflect.Bool:
		return strconv.FormatBool(field.Bool())
	case reflect.Map:
		var result strings.Builder
		for _, key := range field.MapKeys() {
			value := field.MapIndex(key)
			result.WriteString(
				fmt.Sprintf("%s=%s;", key.String(), value.String()),
			)
		}
		return result.String()
	default:
		return fmt.Sprintf("%v", field.Interface())
	}
}
