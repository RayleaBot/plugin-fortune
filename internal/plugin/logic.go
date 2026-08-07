package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

var (
	fixedTimezonePattern = regexp.MustCompile(`(?i)^(?:UTC)?([+-])(\d{1,2})(?::?(\d{2}))?$`)
	numericIDPattern     = regexp.MustCompile(`^\d+$`)
	countedFortunes      = []string{"大吉", "吉", "中吉", "小吉", "末吉", "凶", "大凶", "吉凶未定"}
	fortuneStars         = map[string]string{
		"大吉": "★★★★★★★", "吉": "★★★★★★☆", "中吉": "★★★★★☆☆", "小吉": "★★★★☆☆☆",
		"末吉": "★★★☆☆☆☆", "凶": "★★☆☆☆☆☆", "大凶": "☆☆☆☆☆☆☆", "吉凶未定": "???????",
	}
)

type specialDate struct {
	Date        string   `json:"date"`
	FortuneName string   `json:"fortune_name,omitempty"`
	Fortune     *fortune `json:"fortune,omitempty"`
}

func (item *specialDate) UnmarshalJSON(data []byte) error {
	var raw struct {
		Date        string          `json:"date"`
		FortuneName string          `json:"fortune_name"`
		Fortune     json.RawMessage `json:"fortune"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	item.Date = strings.TrimSpace(raw.Date)
	item.FortuneName = strings.TrimSpace(raw.FortuneName)
	if len(raw.Fortune) == 0 || string(raw.Fortune) == "null" {
		return nil
	}
	var name string
	if json.Unmarshal(raw.Fortune, &name) == nil {
		if item.FortuneName == "" {
			item.FortuneName = strings.TrimSpace(name)
		}
		return nil
	}
	var value fortune
	if err := json.Unmarshal(raw.Fortune, &value); err != nil {
		return err
	}
	item.Fortune = &value
	if item.FortuneName == "" {
		item.FortuneName = value.Name
	}
	return nil
}

type fortuneStats struct {
	TotalDays            int            `json:"total_days"`
	CurrentStreak        int            `json:"current_streak"`
	LastDate             string         `json:"last_date"`
	Counts               map[string]int `json:"counts"`
	CurrentDajiStreak    int            `json:"current_daji_streak"`
	LongestDajiStreak    int            `json:"longest_daji_streak"`
	CurrentDaxiongStreak int            `json:"current_daxiong_streak"`
	LongestDaxiongStreak int            `json:"longest_daxiong_streak"`
}

func emptyStats() fortuneStats {
	counts := make(map[string]int, len(countedFortunes))
	for _, name := range countedFortunes {
		counts[name] = 0
	}
	return fortuneStats{Counts: counts}
}

func normalizeStats(value fortuneStats) fortuneStats {
	if value.TotalDays < 0 {
		value.TotalDays = 0
	}
	if value.CurrentStreak < 0 {
		value.CurrentStreak = 0
	}
	if value.CurrentDajiStreak < 0 {
		value.CurrentDajiStreak = 0
	}
	if value.LongestDajiStreak < 0 {
		value.LongestDajiStreak = 0
	}
	if value.CurrentDaxiongStreak < 0 {
		value.CurrentDaxiongStreak = 0
	}
	if value.LongestDaxiongStreak < 0 {
		value.LongestDaxiongStreak = 0
	}
	counts := make(map[string]int, len(countedFortunes))
	for _, name := range countedFortunes {
		count := value.Counts[name]
		if count < 0 {
			count = 0
		}
		counts[name] = count
	}
	value.Counts = counts
	return value
}

func drawSourceFingerprint(current settings) string {
	payload := struct {
		Fortunes     []fortune     `json:"fortunes"`
		SpecialDates []specialDate `json:"special_dates"`
		GoodActions  []string      `json:"good_actions"`
		BadActions   []string      `json:"bad_actions"`
	}{current.Fortunes, current.SpecialDates, cleanStrings(current.GoodActions), cleanStrings(current.BadActions)}
	raw, _ := json.Marshal(payload)
	digest := sha256.Sum256(raw)
	return fmt.Sprintf("%x", digest[:])
}

func drawDaily(current settings, userID, day string) dailyRecord {
	selected, special := fortuneForDay(current, userID, day)
	good := choose(current.GoodActions, 2, userID, day, "good")
	badPool := make([]string, 0, len(current.BadActions))
	for _, item := range current.BadActions {
		if !contains(good, strings.TrimSpace(item)) {
			badPool = append(badPool, item)
		}
	}
	return dailyRecord{
		Date: day, DrawSourceFingerprint: drawSourceFingerprint(current), Fortune: selected,
		TodayGood: good, TodayBad: choose(badPool, 2, userID, day, "bad"), Special: special,
	}
}

func fortuneForDay(current settings, userID, day string) (fortune, bool) {
	for _, key := range []string{day, monthDay(day)} {
		for _, special := range current.SpecialDates {
			if strings.TrimSpace(special.Date) != key {
				continue
			}
			if special.Fortune != nil && validFortune(*special.Fortune) {
				return *special.Fortune, true
			}
			for _, candidate := range current.Fortunes {
				if candidate.Name == special.FortuneName {
					return candidate, true
				}
			}
		}
	}
	drawable := make([]fortune, 0, len(current.Fortunes))
	for _, candidate := range current.Fortunes {
		if candidate.Name != "吉凶未定" {
			drawable = append(drawable, candidate)
		}
	}
	if len(drawable) == 0 {
		drawable = current.Fortunes
	}
	return drawable[stableIndex(len(drawable), userID, day, "fortune")], false
}

func monthDay(day string) string {
	if len(day) == len("2006-01-02") {
		return day[5:]
	}
	return ""
}

func stableIndex(size int, parts ...string) int {
	if size <= 0 {
		return 0
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return int(binary.BigEndian.Uint64(digest[:8]) % uint64(size))
}

func choose(values []string, count int, parts ...string) []string {
	clean := cleanStrings(values)
	result := make([]string, 0, count)
	for len(clean) > 0 && len(result) < count {
		index := stableIndex(len(clean), append(parts, strconv.Itoa(len(result)))...)
		result = append(result, clean[index])
		clean = append(clean[:index], clean[index+1:]...)
	}
	return result
}

func cleanStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func contains(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func localDay(now time.Time, timezone string) string {
	return now.In(resolveLocation(timezone)).Format("2006-01-02")
}

func resolveLocation(name string) *time.Location {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Asia/Shanghai"
	}
	if location, err := time.LoadLocation(name); err == nil {
		return location
	}
	match := fixedTimezonePattern.FindStringSubmatch(name)
	if len(match) == 4 {
		hours, _ := strconv.Atoi(match[2])
		minutes, _ := strconv.Atoi(match[3])
		if hours <= 14 && minutes <= 59 && !(hours == 14 && minutes != 0) {
			offset := hours*60*60 + minutes*60
			if match[1] == "-" {
				offset = -offset
			}
			return time.FixedZone(name, offset)
		}
	}
	location, err := time.LoadLocation("Asia/Shanghai")
	if err == nil {
		return location
	}
	return time.FixedZone("Asia/Shanghai", 8*60*60)
}

func updateStats(stats fortuneStats, fortuneName, day string) fortuneStats {
	stats = normalizeStats(stats)
	currentDay, _ := time.Parse("2006-01-02", day)
	previousDay, previousValid := parseDay(stats.LastDate)
	if previousValid && previousDay.Equal(currentDay.AddDate(0, 0, -1)) {
		stats.CurrentStreak++
	} else if !previousValid || !previousDay.Equal(currentDay) {
		stats.CurrentStreak = 1
	}
	if !previousValid || !previousDay.Equal(currentDay) {
		stats.TotalDays++
		stats.LastDate = day
	}
	if _, exists := stats.Counts[fortuneName]; exists {
		stats.Counts[fortuneName]++
	}
	if fortuneName == "大吉" {
		stats.CurrentDajiStreak++
	} else {
		stats.CurrentDajiStreak = 0
	}
	if stats.CurrentDajiStreak > stats.LongestDajiStreak {
		stats.LongestDajiStreak = stats.CurrentDajiStreak
	}
	if fortuneName == "大凶" {
		stats.CurrentDaxiongStreak++
	} else {
		stats.CurrentDaxiongStreak = 0
	}
	if stats.CurrentDaxiongStreak > stats.LongestDaxiongStreak {
		stats.LongestDaxiongStreak = stats.CurrentDaxiongStreak
	}
	return stats
}

func replaceCurrentDayFortune(stats fortuneStats, previousName, nextName string) fortuneStats {
	stats = normalizeStats(stats)
	if previousName == nextName {
		return stats
	}
	if _, exists := stats.Counts[previousName]; exists && stats.Counts[previousName] > 0 {
		stats.Counts[previousName]--
	}
	if _, exists := stats.Counts[nextName]; exists {
		stats.Counts[nextName]++
	}
	if previousName == "大吉" && nextName != "大吉" && stats.CurrentDajiStreak > 0 {
		stats.CurrentDajiStreak--
	} else if previousName != "大吉" && nextName == "大吉" {
		stats.CurrentDajiStreak++
		if stats.CurrentDajiStreak > stats.LongestDajiStreak {
			stats.LongestDajiStreak = stats.CurrentDajiStreak
		}
	}
	if previousName == "大凶" && nextName != "大凶" && stats.CurrentDaxiongStreak > 0 {
		stats.CurrentDaxiongStreak--
	} else if previousName != "大凶" && nextName == "大凶" {
		stats.CurrentDaxiongStreak++
		if stats.CurrentDaxiongStreak > stats.LongestDaxiongStreak {
			stats.LongestDaxiongStreak = stats.CurrentDaxiongStreak
		}
	}
	return stats
}

func parseDay(value string) (time.Time, bool) {
	parsed, err := time.Parse("2006-01-02", value)
	return parsed, err == nil
}

func readStats(ctx context.Context, event *rayleabot.EventContext, userID string) (fortuneStats, error) {
	result, err := event.Actions().KVGet(ctx, "stats:"+userID)
	if err != nil {
		return fortuneStats{}, err
	}
	stats := emptyStats()
	if result["exists"] != true {
		return stats, nil
	}
	raw, err := json.Marshal(result["value"])
	if err != nil {
		return stats, nil
	}
	if json.Unmarshal(raw, &stats) != nil {
		return emptyStats(), nil
	}
	return normalizeStats(stats), nil
}

func writeStats(ctx context.Context, event *rayleabot.EventContext, userID string, stats fortuneStats) error {
	_, err := event.Actions().KVSet(ctx, "stats:"+userID, normalizeStats(stats))
	return err
}

func renderUser(event *rayleabot.EventContext) map[string]any {
	onebot, _ := event.Event.Payload["onebot"].(map[string]any)
	sender, _ := onebot["sender"].(map[string]any)
	userID := firstText(onebot["user_id"], sender["user_id"], event.Event.Actor.ID)
	if userID == "" {
		userID = "unknown"
	}
	nickname := firstText(sender["nickname"], event.Event.Actor.Nickname, userID, "访客")
	user := map[string]any{
		"id": userID, "nickname": nickname,
		"group_nickname": firstText(sender["card"]),
		"title":          firstText(sender["title"], event.Event.Actor.Role),
		"avatar_url":     "",
	}
	if numericIDPattern.MatchString(userID) {
		user["avatar_url"] = "https://q1.qlogo.cn/g?b=qq&nk=" + userID + "&s=100"
	}
	return user
}

func renderGroup(event *rayleabot.EventContext) map[string]any {
	if event.Event.Target.Type != "group" {
		return map[string]any{}
	}
	name := firstText(event.Event.Target.Name, event.Event.Target.ID)
	if name == "" {
		return map[string]any{}
	}
	return map[string]any{"name": name}
}

func repeatNotice(repeated bool) string {
	if repeated {
		return "今日运势已经抽取过，以下为当日结果。"
	}
	return ""
}

func statsSummary(stats fortuneStats) []map[string]any {
	result := make([]map[string]any, 0, len(countedFortunes)+2)
	for _, name := range countedFortunes {
		result = append(result, map[string]any{"label": "累计" + name, "value": fmt.Sprintf("%d 次", stats.Counts[name])})
	}
	result = append(result,
		map[string]any{"label": "最长连续大吉", "value": fmt.Sprintf("%d 天", stats.LongestDajiStreak)},
		map[string]any{"label": "最长连续大凶", "value": fmt.Sprintf("%d 天", stats.LongestDaxiongStreak)},
	)
	return result
}

func statsRenderData(event *rayleabot.EventContext, current settings, stats fortuneStats) map[string]any {
	distribution := make([]map[string]any, 0, len(countedFortunes))
	totalDraws := 0
	for _, count := range stats.Counts {
		totalDraws += count
	}
	for _, name := range countedFortunes {
		count := stats.Counts[name]
		if name != "吉凶未定" && count == 0 && totalDraws == 0 {
			continue
		}
		if name == "吉凶未定" && count == 0 && totalDraws > 0 {
			continue
		}
		percentage := float64(0)
		if totalDraws > 0 {
			percentage = math.Round(float64(count)*1000/float64(totalDraws)) / 10
		}
		distribution = append(distribution, map[string]any{
			"name": name, "count": count, "percentage": percentage, "stars": fortuneStars[name],
		})
	}
	summary := make([]map[string]any, 0, 2)
	if stats.TotalDays > 0 {
		summary = append(summary, map[string]any{"label": "累计查看", "value": fmt.Sprintf("%d 天", stats.TotalDays)})
	}
	if stats.CurrentStreak > 0 {
		summary = append(summary, map[string]any{"label": "当前连续", "value": fmt.Sprintf("%d 天", stats.CurrentStreak)})
	}
	records := make([]map[string]any, 0, 2)
	if stats.LongestDajiStreak > 0 {
		records = append(records, map[string]any{"label": "最长连续大吉", "value": fmt.Sprintf("%d 天", stats.LongestDajiStreak)})
	}
	if stats.LongestDaxiongStreak > 0 {
		records = append(records, map[string]any{"label": "最长连续大凶", "value": fmt.Sprintf("%d 天", stats.LongestDaxiongStreak)})
	}
	return map[string]any{
		"title": "运势统计", "subtitle": localDay(time.Now(), current.Timezone),
		"user": renderUser(event), "group": renderGroup(event), "summary": summary,
		"distribution": distribution,
		"records":      records,
	}
}

func formatStats(data map[string]any) string {
	lines := []string{"运势统计"}
	for _, item := range mapSlice(data["summary"]) {
		lines = append(lines, firstText(item["label"])+"："+firstText(item["value"]))
	}
	lines = append(lines, "", "运势分布：")
	for _, item := range mapSlice(data["distribution"]) {
		lines = append(lines, fmt.Sprintf("  %s：%v 次（%.1f%%）", firstText(item["name"]), item["count"], floatValue(item["percentage"])))
	}
	if records := mapSlice(data["records"]); len(records) > 0 {
		lines = append(lines, "", "连续记录：")
		for _, item := range records {
			lines = append(lines, "  "+firstText(item["label"])+"："+firstText(item["value"]))
		}
	}
	return strings.Join(lines, "\n")
}

func firstText(values ...any) string {
	for _, value := range values {
		text := strings.TrimSpace(fmt.Sprint(value))
		if value != nil && text != "" && text != "<nil>" {
			return text
		}
	}
	return ""
}

func mapSlice(value any) []map[string]any {
	if typed, ok := value.([]map[string]any); ok {
		return typed
	}
	return nil
}

func floatValue(value any) float64 {
	result, _ := value.(float64)
	return result
}
