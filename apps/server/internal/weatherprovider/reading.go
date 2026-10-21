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

type forecastResponse struct {
	Timezone     string            `json:"timezone"`
	CurrentUnits map[string]string `json:"current_units"`
	DailyUnits   map[string]string `json:"daily_units"`
	Current      struct {
		Time            int64    `json:"time"`
		TemperatureC    *float64 `json:"temperature_2m"`
		FeelsLikeC      *float64 `json:"apparent_temperature"`
		HumidityPercent *float64 `json:"relative_humidity_2m"`
		WindSpeedMS     *float64 `json:"wind_speed_10m"`
		WeatherCode     *int     `json:"weather_code"`
		IsDay           *int     `json:"is_day"`
	} `json:"current"`
	Daily struct {
		Time              []int64    `json:"time"`
		MinimumC          []*float64 `json:"temperature_2m_min"`
		MaximumC          []*float64 `json:"temperature_2m_max"`
		RainChancePercent []*float64 `json:"precipitation_probability_max"`
		WeatherCode       []*int     `json:"weather_code"`
	} `json:"daily"`
}

func normalize(raw forecastResponse, settings Settings, now time.Time) (Reading, error) {
	location, err := time.LoadLocation(raw.Timezone)
	if err != nil || raw.Timezone == "" || raw.Timezone == "Local" {
		return Reading{}, fmt.Errorf("Open-Meteo returned an invalid timezone %q", raw.Timezone)
	}

	if raw.CurrentUnits["temperature_2m"] != "°C" ||
		raw.CurrentUnits["apparent_temperature"] != "°C" ||
		raw.CurrentUnits["relative_humidity_2m"] != "%" ||
		raw.CurrentUnits["wind_speed_10m"] != "m/s" ||
		raw.DailyUnits["temperature_2m_min"] != "°C" ||
		raw.DailyUnits["temperature_2m_max"] != "°C" ||
		raw.DailyUnits["precipitation_probability_max"] != "%" {
		return Reading{}, fmt.Errorf("Open-Meteo returned unexpected weather units")
	}

	observed := time.Unix(raw.Current.Time, 0)
	age := now.Sub(observed)
	if raw.Current.Time <= 0 || age < -5*time.Minute || age >= settings.TTL {
		return Reading{}, fmt.Errorf("Open-Meteo current weather has an expired or future timestamp")
	}

	current, err := normalizeCurrent(raw)
	if err != nil {
		return Reading{}, err
	}

	days, err := normalizeDays(raw, now.In(location), location)
	if err != nil {
		return Reading{}, err
	}

	return Reading{
		Version:     1,
		Label:       settings.Label,
		Timezone:    raw.Timezone,
		UpdatedAtMS: strconv.FormatInt(observed.UnixMilli(), 10),
		TTLSeconds:  int64(settings.TTL / time.Second),
		Current:     current,
		Days:        days,
	}, nil
}

func normalizeCurrent(raw forecastResponse) (Current, error) {
	current := raw.Current
	if !validNumber(current.TemperatureC, -100, 70) ||
		!validNumber(current.FeelsLikeC, -120, 90) ||
		!validNumber(current.HumidityPercent, 0, 100) ||
		!validNumber(current.WindSpeedMS, 0, 150) ||
		current.IsDay == nil || (*current.IsDay != 0 && *current.IsDay != 1) ||
		current.WeatherCode == nil || condition(*current.WeatherCode) == "" {
		return Current{}, fmt.Errorf("Open-Meteo current weather is incomplete or invalid")
	}

	return Current{
		TemperatureC:    *current.TemperatureC,
		FeelsLikeC:      *current.FeelsLikeC,
		HumidityPercent: *current.HumidityPercent,
		WindSpeedMS:     *current.WindSpeedMS,
		WeatherCode:     *current.WeatherCode,
		Condition:       condition(*current.WeatherCode),
		IsDay:           *current.IsDay == 1,
	}, nil
}

func normalizeDays(raw forecastResponse, today time.Time, location *time.Location) ([]Day, error) {
	daily := raw.Daily
	count := len(daily.Time)
	if count != 4 || len(daily.MinimumC) != count || len(daily.MaximumC) != count ||
		len(daily.RainChancePercent) != count || len(daily.WeatherCode) != count {
		return nil, fmt.Errorf("Open-Meteo daily forecast must contain four complete days")
	}

	days := make([]Day, 0, 3)
	for index := range daily.Time {
		date := time.Unix(daily.Time[index], 0).In(location)
		if date.Format("2006-01-02") != today.AddDate(0, 0, index).Format("2006-01-02") ||
			!validNumber(daily.MinimumC[index], -100, 70) ||
			!validNumber(daily.MaximumC[index], -100, 70) ||
			*daily.MinimumC[index] > *daily.MaximumC[index] ||
			!validNumber(daily.RainChancePercent[index], 0, 100) ||
			daily.WeatherCode[index] == nil || condition(*daily.WeatherCode[index]) == "" {
			return nil, fmt.Errorf("Open-Meteo forecast day %d is incomplete or invalid", index)
		}

		if index == 0 {
			continue
		}

		days = append(days, Day{
			Date:                            date.Format("2006-01-02"),
			Weekday:                         date.Format("Mon"),
			MinimumC:                        *daily.MinimumC[index],
			MaximumC:                        *daily.MaximumC[index],
			PrecipitationProbabilityPercent: *daily.RainChancePercent[index],
			WeatherCode:                     *daily.WeatherCode[index],
			Condition:                       condition(*daily.WeatherCode[index]),
		})
	}

	return days, nil
}

func validNumber(value *float64, minimum, maximum float64) bool {
	return value != nil && inRange(*value, minimum, maximum)
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
