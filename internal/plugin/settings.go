package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-fortune/internal/assets"
)

var (
	fullDatePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	monthDayPattern = regexp.MustCompile(`^\d{2}-\d{2}$`)
)

func loadSettings(ctx context.Context, event *rayleabot.EventContext) (settings, error) {
	defaults, err := defaultSettings()
	if err != nil {
		return settings{}, err
	}
	result, err := event.Actions().ConfigRead(ctx, "trigger_commands", "stats_trigger_commands", "timezone", "fortunes", "special_dates", "good_actions", "bad_actions")
	if err != nil {
		logFortune(ctx, event.Actions(), "warn", "运势设置读取失败，使用默认设置", map[string]any{"error": err.Error()})
		return defaults, nil
	}
	values, _ := result["values"].(map[string]any)
	for _, message := range settingsIssueMessages(values) {
		logFortune(ctx, event.Actions(), "warn", message, nil)
	}
	return mergeSettings(defaults, values), nil
}

func defaultSettings() (settings, error) {
	var raw map[string]any
	if err := json.Unmarshal(assets.DefaultSettingsJSON, &raw); err != nil {
		return settings{}, fmt.Errorf("decode embedded fortune settings: %w", err)
	}
	current := normalizeSettings(raw, settings{})
	if len(current.Fortunes) == 0 {
		return settings{}, fmt.Errorf("embedded fortune list has no valid entries")
	}
	return current, nil
}

func mergeSettings(defaults settings, values map[string]any) settings {
	if values == nil {
		return defaults
	}
	raw := settingsMap(defaults)
	for _, key := range []string{"trigger_commands", "stats_trigger_commands", "timezone", "fortunes", "special_dates", "good_actions", "bad_actions"} {
		if value, exists := values[key]; exists {
			raw[key] = value
		}
	}
	return normalizeSettings(raw, defaults)
}

func normalizeSettings(raw map[string]any, defaults settings) settings {
	current := settings{
		TriggerCommands:      normalizeStringList(raw["trigger_commands"], fallbackStrings(defaults.TriggerCommands, []string{"我的运势"})),
		StatsTriggerCommands: normalizeStringList(raw["stats_trigger_commands"], fallbackStrings(defaults.StatsTriggerCommands, []string{"运势统计"})),
		Timezone:             normalizeTimezone(stringFromAny(raw["timezone"])),
		Fortunes:             normalizeFortunes(raw["fortunes"]),
		SpecialDates:         normalizeSpecialDates(raw["special_dates"]),
		GoodActions:          normalizeStringList(raw["good_actions"], fallbackStrings(defaults.GoodActions, []string{"整理计划"})),
		BadActions:           normalizeStringList(raw["bad_actions"], fallbackStrings(defaults.BadActions, []string{"熬夜"})),
	}
	if len(current.Fortunes) == 0 && len(defaults.Fortunes) > 0 {
		current.Fortunes = append([]fortune(nil), defaults.Fortunes...)
	}
	return current
}

func normalizeFortunes(value any) []fortune {
	items, ok := value.([]any)
	if !ok {
		raw, _ := json.Marshal(value)
		var typed []fortune
		if json.Unmarshal(raw, &typed) == nil {
			result := make([]fortune, 0, len(typed))
			for _, item := range typed {
				if normalized, ok := normalizeFortune(item); ok {
					result = append(result, normalized)
				}
			}
			return result
		}
		return nil
	}
	result := make([]fortune, 0, len(items))
	for _, item := range items {
		raw, _ := json.Marshal(item)
		var candidate fortune
		if json.Unmarshal(raw, &candidate) != nil {
			continue
		}
		if normalized, ok := normalizeFortune(candidate); ok {
			result = append(result, normalized)
		}
	}
	return result
}

func normalizeFortune(value fortune) (fortune, bool) {
	value.Name = strings.TrimSpace(value.Name)
	value.Stars = strings.TrimSpace(value.Stars)
	value.Sign = strings.TrimSpace(value.Sign)
	value.Explanation = strings.TrimSpace(value.Explanation)
	return value, validFortune(value)
}

func normalizeSpecialDates(value any) []specialDate {
	raw, _ := json.Marshal(value)
	var items []specialDate
	if json.Unmarshal(raw, &items) != nil {
		return nil
	}
	result := make([]specialDate, 0, len(items))
	for _, item := range items {
		item.Date = strings.TrimSpace(item.Date)
		item.FortuneName = strings.TrimSpace(item.FortuneName)
		if !validSpecialDate(item.Date) {
			continue
		}
		if item.Fortune != nil {
			if normalized, ok := normalizeFortune(*item.Fortune); ok {
				item.Fortune = &normalized
				if item.FortuneName == "" {
					item.FortuneName = normalized.Name
				}
			} else {
				item.Fortune = nil
			}
		}
		if item.FortuneName == "" && item.Fortune == nil {
			continue
		}
		result = append(result, item)
	}
	return result
}

func normalizeStringList(value any, fallback []string) []string {
	items, ok := value.([]any)
	if !ok {
		if typed, typedOK := value.([]string); typedOK {
			items = make([]any, len(typed))
			for index := range typed {
				items[index] = typed[index]
			}
		} else {
			return append([]string(nil), fallback...)
		}
	}
	result := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		text := strings.TrimSpace(stringFromAny(item))
		if text == "" {
			continue
		}
		if _, exists := seen[text]; exists {
			continue
		}
		seen[text] = struct{}{}
		result = append(result, text)
	}
	if len(result) == 0 {
		return append([]string(nil), fallback...)
	}
	return result
}

func validFortune(value fortune) bool {
	if value.Name == "" || value.Sign == "" || value.Explanation == "" || len([]rune(value.Stars)) != 7 {
		return false
	}
	if value.Name == "凶" {
		return value.Stars == "★☆☆☆☆☆☆" || value.Stars == "★★☆☆☆☆☆"
	}
	expected, exists := fortuneStars[value.Name]
	return exists && value.Stars == expected
}

func validSpecialDate(value string) bool {
	if fullDatePattern.MatchString(value) {
		parsed, err := time.Parse("2006-01-02", value)
		return err == nil && parsed.Format("2006-01-02") == value
	}
	if monthDayPattern.MatchString(value) {
		parsed, err := time.Parse("2006-01-02", "2024-"+value)
		return err == nil && parsed.Format("01-02") == value
	}
	return false
}

func normalizeTimezone(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Asia/Shanghai"
	}
	if _, err := time.LoadLocation(value); err == nil {
		return value
	}
	if fixedTimezonePattern.MatchString(value) {
		match := fixedTimezonePattern.FindStringSubmatch(value)
		hours := intValueText(match[2])
		minutes := intValueText(match[3])
		if hours <= 14 && minutes <= 59 && !(hours == 14 && minutes != 0) {
			return value
		}
	}
	return "Asia/Shanghai"
}

func settingsIssueMessages(values map[string]any) []string {
	if values == nil {
		return nil
	}
	var result []string
	if raw, exists := values["timezone"]; exists {
		name := strings.TrimSpace(stringFromAny(raw))
		if name != "" && normalizeTimezone(name) != name {
			result = append(result, "运势时区无效，使用默认时区")
		}
	}
	if raw, exists := values["fortunes"]; exists {
		items, ok := raw.([]any)
		if ok && len(items) > 0 {
			valid := len(normalizeFortunes(raw))
			if valid == 0 {
				result = append(result, "运势覆盖没有可用条目，使用默认运势库")
			} else if valid < len(items) {
				result = append(result, "部分运势条目无效，已跳过")
			}
		}
	}
	if raw, exists := values["special_dates"]; exists {
		if items, ok := raw.([]any); ok && len(normalizeSpecialDates(raw)) < len(items) {
			result = append(result, "部分特殊日期无效，已跳过")
		}
	}
	return result
}

func settingsMap(value settings) map[string]any {
	raw, _ := json.Marshal(value)
	var result map[string]any
	_ = json.Unmarshal(raw, &result)
	return result
}

func fallbackStrings(primary, fallback []string) []string {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

func stringFromAny(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func intValueText(value string) int {
	var result int
	_, _ = fmt.Sscan(value, &result)
	return result
}

func logFortune(ctx context.Context, actions interface {
	LoggerWrite(context.Context, rayleabot.LoggerWriteRequest) (rayleabot.ActionResult, error)
}, level, message string, fields map[string]any) {
	_, _ = actions.LoggerWrite(ctx, rayleabot.LoggerWriteRequest{Level: level, Message: message, Fields: fields})
}
