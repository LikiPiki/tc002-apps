package clockapps

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var attributes = regexp.MustCompile(`([a-zA-Z_]+)\s*=\s*("[^"]*"|[^\s]+)`)
var settingKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,23}$`)
var clockTime = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

type field struct {
	typ   string
	attrs map[string]string
}

func validateSettings(source []byte, settings map[string]any) error {
	declared, err := parseFields(source)
	if err != nil {
		return err
	}

	for key, value := range settings {
		field, ok := declared[key]
		if !ok {
			return fmt.Errorf("setting %q has no @config declaration", key)
		}
		if err := field.validate(value); err != nil {
			return fmt.Errorf("setting %q: %w", key, err)
		}
	}

	return nil
}

func parseFields(source []byte) (map[string]field, error) {
	declared := map[string]field{}
	for _, line := range strings.Split(string(source), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "#") {
			break
		}

		text := strings.TrimSpace(strings.TrimPrefix(line, "#"))
		words := strings.Fields(text)
		if len(words) == 0 || words[0] != "@config" {
			continue
		}
		if len(words) < 3 || !settingKey.MatchString(words[1]) {
			return nil, fmt.Errorf("invalid @config declaration: %s", text)
		}

		key := words[1]
		if _, ok := declared[key]; ok {
			return nil, fmt.Errorf("duplicate @config %s", key)
		}

		field, err := parseField(text, words[2])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}

		declared[key] = field
	}

	return declared, nil
}

func parseField(declaration, fieldType string) (field, error) {
	fieldType = strings.ToLower(fieldType)
	switch fieldType {
	case "bool", "text", "select", "number", "slider", "color":
	default:
		return field{}, fmt.Errorf("unknown @config type %s", fieldType)
	}

	parsed := field{
		typ:   fieldType,
		attrs: map[string]string{},
	}

	for _, match := range attributes.FindAllStringSubmatch(declaration, -1) {
		value := match[2]
		if strings.HasPrefix(value, "\"") {
			value = strings.Trim(value, "\"")
		}
		parsed.attrs[strings.ToLower(match[1])] = value
	}

	if fieldType == "slider" && (parsed.attrs["min"] == "" || parsed.attrs["max"] == "") {
		parsed.attrs["min"] = "0"
		parsed.attrs["max"] = "100"
	}

	if err := parsed.validateAttributes(); err != nil {
		return field{}, err
	}
	if err := parsed.validateDefault(); err != nil {
		return field{}, err
	}

	return parsed, nil
}

func (f field) validateAttributes() error {
	for _, attribute := range []string{"min", "max", "step", "maxlen"} {
		raw, exists := f.attrs[attribute]
		if !exists {
			continue
		}

		value, err := strconv.ParseFloat(raw, 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("invalid %s: expected a finite number", attribute)
		}
		if (attribute == "step" || attribute == "maxlen") && value <= 0 {
			return fmt.Errorf("%s must be positive", attribute)
		}
		if attribute == "maxlen" && value != math.Trunc(value) {
			return fmt.Errorf("maxlen must be an integer")
		}
	}

	minimum, hasMinimum := f.attrs["min"]
	maximum, hasMaximum := f.attrs["max"]
	if hasMinimum && hasMaximum {
		lower, _ := strconv.ParseFloat(minimum, 64)
		upper, _ := strconv.ParseFloat(maximum, 64)
		if lower > upper {
			return fmt.Errorf("min exceeds max")
		}
	}

	if f.typ == "select" && f.attrs["options"] == "" {
		return fmt.Errorf("select requires options")
	}

	return nil
}

func (f field) validateDefault() error {
	raw, exists := f.attrs["default"]
	if !exists {
		return nil
	}

	var value any = raw
	switch f.typ {
	case "bool":
		switch strings.ToLower(raw) {
		case "true", "1", "yes", "on":
			value = true
		case "false", "0", "no", "off":
			value = false
		default:
			return fmt.Errorf("invalid boolean default")
		}
	case "number", "slider":
		number, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return fmt.Errorf("invalid numeric default")
		}

		value = number
	}

	if err := f.validate(value); err != nil {
		return fmt.Errorf("default: %w", err)
	}

	return nil
}

func (f field) validate(value any) error {
	bad := fmt.Errorf("must match @config type %s and its constraints", f.typ)
	switch f.typ {
	case "bool":
		if _, ok := value.(bool); !ok {
			return bad
		}
	case "text", "select":
		s, ok := value.(string)
		if !ok {
			return bad
		}

		limit := 256
		if v, ok := f.attrs["maxlen"]; ok {
			limit, _ = strconv.Atoi(v)
		}
		if len(s) > limit {
			return fmt.Errorf("exceeds maxlen=%d bytes", limit)
		}

		if f.typ == "select" {
			for _, option := range strings.Split(f.attrs["options"], ",") {
				if s == strings.TrimSpace(option) {
					return nil
				}
			}

			return fmt.Errorf("must be one of %s", f.attrs["options"])
		}
	case "number", "slider":
		var n float64
		switch v := value.(type) {
		case int:
			n = float64(v)
		case int64:
			n = float64(v)
		case float64:
			n = v
		default:
			return bad
		}

		if math.IsNaN(n) || math.IsInf(n, 0) {
			return bad
		}

		if raw, ok := f.attrs["min"]; ok {
			lo, _ := strconv.ParseFloat(raw, 64)
			if n < lo {
				return fmt.Errorf("below min=%s", raw)
			}
		}

		if raw, ok := f.attrs["max"]; ok {
			hi, _ := strconv.ParseFloat(raw, 64)
			if n > hi {
				return fmt.Errorf("above max=%s", raw)
			}
		}
	case "color":
		if _, err := Color(value); err != nil {
			return err
		}
	}

	return nil
}

// Color accepts the same RGB encodings as AWTRIX and returns its canonical value.
func Color(value any) (uint32, error) {
	var n int64
	switch v := value.(type) {
	case int:
		n = int64(v)
	case int64:
		n = v
	case string:
		base := 10
		if strings.HasPrefix(v, "#") {
			v = v[1:]
			base = 16
			if len(v) != 6 {
				return 0, fmt.Errorf("color requires six hex digits")
			}
		} else if strings.HasPrefix(strings.ToLower(v), "0x") {
			v = v[2:]
			base = 16
			if len(v) != 6 {
				return 0, fmt.Errorf("color requires six hex digits")
			}
		}

		var err error
		n, err = strconv.ParseInt(v, base, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid RGB color")
		}
	default:
		return 0, fmt.Errorf("invalid RGB color type")
	}

	if n < 0 || n > 0xFFFFFF {
		return 0, fmt.Errorf("RGB color outside 0..0xFFFFFF")
	}

	return uint32(n), nil
}

func validateDomain(entry Script) error {
	if entry.Kind != "services" || entry.Name != "display-policy" {
		return nil
	}

	on, off := "10:00", "20:00"
	if v, ok := entry.Settings["on"]; ok {
		on, _ = v.(string)
	}
	if v, ok := entry.Settings["off"]; ok {
		off, _ = v.(string)
	}

	if !clockTime.MatchString(on) || !clockTime.MatchString(off) {
		return fmt.Errorf("display-policy: on and off must be valid HH:MM times")
	}
	if on == off {
		return fmt.Errorf("display-policy: on and off must differ")
	}

	return nil
}
