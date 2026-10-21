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
			return fmt.Errorf("invalid @config declaration: %s", text)
		}
		key, typ := words[1], strings.ToLower(words[2])
		if _, ok := declared[key]; ok {
			return fmt.Errorf("duplicate @config %s", key)
		}
		switch typ {
		case "bool", "text", "select", "number", "slider", "color":
		default:
			return fmt.Errorf("unknown @config type %s", typ)
		}
		f := field{typ, map[string]string{}}
		for _, match := range attributes.FindAllStringSubmatch(text, -1) {
			v := match[2]
			if strings.HasPrefix(v, "\"") {
				v = strings.Trim(v, "\"")
			}
			f.attrs[strings.ToLower(match[1])] = v
		}
		if typ == "slider" && (f.attrs["min"] == "" || f.attrs["max"] == "") {
			f.attrs["min"], f.attrs["max"] = "0", "100"
		}
		for _, attr := range []string{"min", "max", "step", "maxlen"} {
			if raw, ok := f.attrs[attr]; ok {
				v, err := strconv.ParseFloat(raw, 64)
				if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || ((attr == "step" || attr == "maxlen") && v <= 0) || (attr == "maxlen" && v != math.Trunc(v)) {
					return fmt.Errorf("%s: invalid %s", key, attr)
				}
			}
		}
		if lo, ok := f.attrs["min"]; ok {
			if hi, ok := f.attrs["max"]; ok {
				a, _ := strconv.ParseFloat(lo, 64)
				b, _ := strconv.ParseFloat(hi, 64)
				if a > b {
					return fmt.Errorf("%s: min exceeds max", key)
				}
			}
		}
		if typ == "select" && f.attrs["options"] == "" {
			return fmt.Errorf("%s: select requires options", key)
		}
		declared[key] = f
		if raw, ok := f.attrs["default"]; ok {
			var v any = raw
			switch typ {
			case "bool":
				switch strings.ToLower(raw) {
				case "true", "1", "yes", "on":
					v = true
				case "false", "0", "no", "off":
					v = false
				default:
					return fmt.Errorf("%s: invalid boolean default", key)
				}
			case "number", "slider":
				n, err := strconv.ParseFloat(raw, 64)
				if err != nil {
					return fmt.Errorf("%s: invalid numeric default", key)
				}
				v = n
			}
			if err := f.validate(v); err != nil {
				return fmt.Errorf("%s default: %w", key, err)
			}
		}
	}
	for key, value := range settings {
		f, ok := declared[key]
		if !ok {
			return fmt.Errorf("setting %q has no @config declaration", key)
		}
		if err := f.validate(value); err != nil {
			return fmt.Errorf("setting %q: %w", key, err)
		}
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
