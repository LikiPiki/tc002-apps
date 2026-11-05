package weatherprovider

import (
	"fmt"
	"strconv"
	"time"
)

type Reading struct {
	Version     int     `json:"version"`
	Label       string  `json:"label"`
	Timezone    string  `json:"timezone"`
	UpdatedAtMS string  `json:"updated_at_ms"`
	TTLSeconds  int64   `json:"ttl_seconds"`
	Current     Current `json:"current"`
	Today       Day     `json:"today"`
	Days        []Day   `json:"days"`
}

type Current struct {
	TemperatureC    float64 `json:"temperature_c"`
	FeelsLikeC      float64 `json:"feels_like_c"`
	HumidityPercent float64 `json:"humidity_percent"`
	WindSpeedMS     float64 `json:"wind_speed_ms"`
	WeatherCode     int     `json:"weather_code"`
	Condition       string  `json:"condition"`
	IsDay           bool    `json:"is_day"`
}

type Day struct {
	Date                            string  `json:"date"`
	Weekday                         string  `json:"weekday"`
	MinimumC                        float64 `json:"minimum_c"`
	MaximumC                        float64 `json:"maximum_c"`
	PrecipitationProbabilityPercent float64 `json:"precipitation_probability_percent"`
	WeatherCode                     int     `json:"weather_code"`
	Condition                       string  `json:"condition"`
}

type wttrConditions struct {
	Temperature string `json:"temp_C"`
	FeelsLike   string `json:"FeelsLikeC"`
	Humidity    string `json:"humidity"`
	Wind        string `json:"windspeedKmph"`
	Code        string `json:"weatherCode"`
	Observed    string `json:"observation_time"`
}

type forecastResponse struct {
	Current []wttrConditions `json:"current_condition"`
	Days    []struct {
		Date      string `json:"date"`
		Minimum   string `json:"mintempC"`
		Maximum   string `json:"maxtempC"`
		Astronomy []struct {
			Sunrise string `json:"sunrise"`
			Sunset  string `json:"sunset"`
		} `json:"astronomy"`
		Hourly []struct {
			Time string `json:"time"`
			Code string `json:"weatherCode"`
			Rain string `json:"chanceofrain"`
			Snow string `json:"chanceofsnow"`
		} `json:"hourly"`
	} `json:"weather"`
}

func normalize(raw forecastResponse, settings Settings, now time.Time) (Reading, error) {
	location, err := time.LoadLocation(settings.Timezone)
	if err != nil || settings.Timezone == "" || settings.Timezone == "Local" {
		return Reading{}, fmt.Errorf("wttr.in requires an explicit IANA timezone")
	}
	if len(raw.Current) != 1 || len(raw.Days) != 3 {
		return Reading{}, fmt.Errorf("wttr.in must return current conditions and three forecast days")
	}
	current := raw.Current[0]
	// wttr.in reports observation_time in UTC without a date. Use the most
	// recent occurrence, including yesterday when the UTC day has just begun.
	clock, err := time.Parse("03:04 PM", current.Observed)
	if err != nil {
		return Reading{}, fmt.Errorf("wttr.in observation time: %w", err)
	}
	utc := now.UTC()
	observed := time.Date(utc.Year(), utc.Month(), utc.Day(), clock.Hour(), clock.Minute(), 0, 0, time.UTC)
	if observed.After(now) {
		observed = observed.AddDate(0, 0, -1)
	}
	if now.Sub(observed) >= settings.TTL {
		return Reading{}, fmt.Errorf("wttr.in current weather has an expired timestamp")
	}

	values := []struct {
		raw          string
		minimum, max float64
	}{
		{current.Temperature, -100, 70},
		{current.FeelsLike, -120, 90},
		{current.Humidity, 0, 100},
		{current.Wind, 0, 540},
	}
	numbers := make([]float64, len(values))
	for i, value := range values {
		numbers[i], err = parseNumber(value.raw, value.minimum, value.max)
		if err != nil {
			return Reading{}, err
		}
	}
	code, err := wttrCode(current.Code)
	if err != nil {
		return Reading{}, err
	}

	local := now.In(location)
	days := make([]Day, 0, len(raw.Days))
	for index, day := range raw.Days {
		date, err := time.ParseInLocation("2006-01-02", day.Date, location)
		if err != nil || day.Date != local.AddDate(0, 0, index).Format("2006-01-02") || len(day.Hourly) != 8 {
			return Reading{}, fmt.Errorf("wttr.in forecast day %d is incomplete or has an invalid date", index)
		}
		low, lowErr := parseNumber(day.Minimum, -100, 70)
		high, highErr := parseNumber(day.Maximum, -100, 70)
		if lowErr != nil || highErr != nil || low > high {
			return Reading{}, fmt.Errorf("wttr.in forecast day %d has an invalid temperature range", index)
		}
		var chance float64
		for hour, entry := range day.Hourly {
			if entry.Time != strconv.Itoa(hour*300) {
				return Reading{}, fmt.Errorf("wttr.in forecast day %d has invalid hourly times", index)
			}
			for _, value := range []string{entry.Rain, entry.Snow} {
				number, err := parseNumber(value, 0, 100)
				if err != nil {
					return Reading{}, err
				}
				chance = max(chance, number)
			}
			if _, err := wttrCode(entry.Code); err != nil {
				return Reading{}, err
			}
		}
		// Use the noon forecast for the daily condition.
		dailyCode, _ := wttrCode(day.Hourly[4].Code)
		days = append(days, Day{
			Date:                            day.Date,
			Weekday:                         date.Format("Mon"),
			MinimumC:                        low,
			MaximumC:                        high,
			PrecipitationProbabilityPercent: chance,
			WeatherCode:                     dailyCode,
			Condition:                       condition(dailyCode),
		})
	}
	if len(raw.Days[0].Astronomy) != 1 {
		return Reading{}, fmt.Errorf("wttr.in is missing sunrise and sunset")
	}
	astronomy := raw.Days[0].Astronomy[0]
	sunrise, riseErr := time.ParseInLocation("2006-01-02 03:04 PM", days[0].Date+" "+astronomy.Sunrise, location)
	sunset, setErr := time.ParseInLocation("2006-01-02 03:04 PM", days[0].Date+" "+astronomy.Sunset, location)
	if riseErr != nil || setErr != nil || !sunrise.Before(sunset) {
		return Reading{}, fmt.Errorf("wttr.in has invalid sunrise or sunset")
	}
	return Reading{
		Version:     1,
		Label:       settings.Label,
		Timezone:    settings.Timezone,
		UpdatedAtMS: strconv.FormatInt(observed.UnixMilli(), 10),
		TTLSeconds:  int64(settings.TTL / time.Second),
		Current: Current{
			TemperatureC:    numbers[0],
			FeelsLikeC:      numbers[1],
			HumidityPercent: numbers[2],
			WindSpeedMS:     numbers[3] / 3.6,
			WeatherCode:     code,
			Condition:       condition(code),
			IsDay:           !local.Before(sunrise) && local.Before(sunset),
		},
		Today: days[0],
		Days:  days[1:],
	}, nil
}

func parseNumber(raw string, minimum, maximum float64) (float64, error) {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || !inRange(value, minimum, maximum) {
		return 0, fmt.Errorf("wttr.in returned an invalid weather number %q", raw)
	}
	return value, nil
}

// Translate WorldWeatherOnline codes to the existing WMO clock contract.
func wttrCode(raw string) (int, error) {
	code, err := strconv.Atoi(raw)
	if err == nil {
		switch code {
		case 113:
			return 0, nil
		case 116:
			return 2, nil
		case 119, 122:
			return 3, nil
		case 143, 248, 260:
			return 45, nil
		case 176, 263, 266:
			return 51, nil
		case 185, 281, 284:
			return 56, nil
		case 293, 296, 353:
			return 61, nil
		case 299, 302, 305, 308, 356, 359:
			return 65, nil
		case 182, 311, 314, 317, 320, 362, 365:
			return 66, nil
		case 179, 227, 230, 323, 326, 329, 332, 335, 338, 350, 368, 371, 374, 377:
			return 73, nil
		case 200, 386, 389, 392, 395:
			return 95, nil
		}
	}
	return 0, fmt.Errorf("wttr.in returned an unknown weather code %q", raw)
}

func condition(code int) string {
	switch code {
	case 0, 1:
		return "CLEAR"
	case 2, 3:
		return "CLOUD"
	case 45, 48:
		return "FOG"
	case 51, 53, 55, 56, 57:
		return "DRIZZLE"
	case 61, 63, 65, 66, 67, 80, 81, 82:
		return "RAIN"
	case 71, 73, 75, 77, 85, 86:
		return "SNOW"
	case 95, 96, 99:
		return "STORM"
	default:
		return ""
	}
}
