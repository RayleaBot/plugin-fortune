package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

type fortune struct {
	Name        string `json:"name"`
	Stars       string `json:"stars"`
	Sign        string `json:"sign"`
	Explanation string `json:"explanation"`
}

type settings struct {
	TriggerCommands      []string      `json:"trigger_commands"`
	StatsTriggerCommands []string      `json:"stats_trigger_commands"`
	Timezone             string        `json:"timezone"`
	Fortunes             []fortune     `json:"fortunes"`
	SpecialDates         []specialDate `json:"special_dates"`
	GoodActions          []string      `json:"good_actions"`
	BadActions           []string      `json:"bad_actions"`
}

type dailyRecord struct {
	Date                  string   `json:"date"`
	DrawSourceFingerprint string   `json:"draw_source_fingerprint"`
	Fortune               fortune  `json:"fortune"`
	TodayGood             []string `json:"today_good"`
	TodayBad              []string `json:"today_bad"`
	Special               bool     `json:"special"`
}

func Run(ctx context.Context) error {
	return rayleabot.Run(ctx, rayleabot.Options{
		PluginID:              "raylea.fortune",
		Subscriptions:         []string{"message.group", "message.private", "config.changed"},
		MaxConcurrentHandlers: 4,
	}, rayleabot.HandlerFunc(handleEvent))
}

func handleEvent(ctx context.Context, event *rayleabot.EventContext) error {
	if event.Event.EventType == "config.changed" {
		return event.Result(map[string]any{"reloaded": true})
	}
	command := strings.TrimSpace(event.Event.Command())
	if command == "" {
		return event.Result(map[string]any{"handled": false})
	}
	current, err := loadSettings(ctx, event)
	if err != nil {
		return err
	}
	switch matchConfiguredCommand(command, current) {
	case "fortune":
		return sendDailyFortune(ctx, event, current)
	case "fortune_stats":
		return sendStats(ctx, event, current)
	default:
		return event.Result(map[string]any{"handled": false})
	}
}

func matchConfiguredCommand(command string, current settings) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}
	if commandMatches(command, current.TriggerCommands) {
		return "fortune"
	}
	if commandMatches(command, current.StatsTriggerCommands) {
		return "fortune_stats"
	}
	return ""
}

func commandMatches(command string, triggers []string) bool {
	for _, trigger := range triggers {
		if strings.TrimSpace(trigger) == command {
			return true
		}
	}
	return false
}

func sendDailyFortune(ctx context.Context, event *rayleabot.EventContext, current settings) error {
	userID := renderUser(event)["id"].(string)
	day := localDay(time.Now(), current.Timezone)
	key := "daily:" + userID + ":" + day
	record, repeated := readDailyRecord(ctx, event, key)
	stats, err := readStats(ctx, event, userID)
	if err != nil {
		return err
	}
	fingerprint := drawSourceFingerprint(current)
	if repeated && record.DrawSourceFingerprint != fingerprint {
		previousName := record.Fortune.Name
		record = drawDaily(current, userID, day)
		stats = replaceCurrentDayFortune(stats, previousName, record.Fortune.Name)
		repeated = false
		if _, err := event.Actions().KVSet(ctx, key, record); err != nil {
			return err
		}
		if err := writeStats(ctx, event, userID, stats); err != nil {
			return err
		}
	} else if !repeated {
		record = drawDaily(current, userID, day)
		if _, err := event.Actions().KVSet(ctx, key, record); err != nil {
			return err
		}
		stats = updateStats(stats, record.Fortune.Name, day)
		if err := writeStats(ctx, event, userID, stats); err != nil {
			return err
		}
	}
	fallback := formatFortune(record, stats, repeated)
	result, err := event.Actions().RenderImage(ctx, rayleabot.RenderImageRequest{
		Template: "card",
		Data: map[string]any{
			"title": "今日运势", "subtitle": record.Date, "fortune": record.Fortune,
			"today_good": record.TodayGood, "today_bad": record.TodayBad,
			"repeat_notice": repeatNotice(repeated), "user": renderUser(event), "group": renderGroup(event),
			"streak": map[string]any{"current": stats.CurrentStreak, "total": stats.TotalDays},
			"stats":  statsSummary(stats),
		},
		Output: "png", FallbackText: fallback,
	})
	if err == nil {
		if imagePath, _ := result["image_path"].(string); imagePath != "" {
			return event.Send(event.Event.Target.Type, event.Event.Target.ID, rayleabot.Image(imagePath))
		}
	}
	return event.SendText(fallback)
}

func sendStats(ctx context.Context, event *rayleabot.EventContext, current settings) error {
	userID := renderUser(event)["id"].(string)
	stats, err := readStats(ctx, event, userID)
	if err != nil {
		return err
	}
	if stats.TotalDays == 0 {
		return event.SendText("你还没有抽取过运势，发送「我的运势」来抽取今日运势吧！")
	}
	data := statsRenderData(event, current, stats)
	fallback := formatStats(data)
	result, renderErr := event.Actions().RenderImage(ctx, rayleabot.RenderImageRequest{
		Template: "stats", Data: data, Output: "png", FallbackText: fallback,
	})
	if renderErr == nil {
		if imagePath, _ := result["image_path"].(string); imagePath != "" {
			return event.Send(event.Event.Target.Type, event.Event.Target.ID, rayleabot.Image(imagePath))
		}
	}
	return event.SendText(fallback)
}

func readDailyRecord(ctx context.Context, event *rayleabot.EventContext, key string) (dailyRecord, bool) {
	result, err := event.Actions().KVGet(ctx, key)
	if err != nil || result["exists"] != true {
		return dailyRecord{}, false
	}
	raw, err := json.Marshal(result["value"])
	if err != nil {
		return dailyRecord{}, false
	}
	var record dailyRecord
	if json.Unmarshal(raw, &record) != nil || record.Date == "" || record.Fortune.Name == "" {
		return dailyRecord{}, false
	}
	return record, true
}

func formatFortune(record dailyRecord, stats fortuneStats, repeated bool) string {
	lines := []string{"今日运势", record.Date}
	if repeated {
		lines = append(lines, "今日已抽取过，以下为原结果。")
	}
	lines = append(lines,
		"运势："+record.Fortune.Name,
		"星级："+record.Fortune.Stars,
		"签文："+record.Fortune.Sign,
		"解签："+record.Fortune.Explanation,
		"今日宜："+strings.Join(record.TodayGood, "、"),
		"今日忌："+strings.Join(record.TodayBad, "、"),
		fmt.Sprintf("你已经连续查看运势 %d 天。累计查看运势 %d 天。", stats.CurrentStreak, stats.TotalDays),
	)
	return strings.Join(lines, "\n")
}
