package plugin

import (
	"reflect"
	"strings"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func TestDrawDailyIsStableAndUnique(t *testing.T) {
	current := settings{
		Fortunes: []fortune{
			{Name: "大吉", Stars: "★★★★★★★", Sign: "a", Explanation: "a"},
			{Name: "中吉", Stars: "★★★★★☆☆", Sign: "b", Explanation: "b"},
			{Name: "吉凶未定", Stars: "???????", Sign: "c", Explanation: "c"},
		},
		GoodActions: []string{"读书", "散步", "整理"},
		BadActions:  []string{"熬夜", "冲动", "拖延", "读书"},
	}
	first := drawDaily(current, "42", "2026-08-02")
	second := drawDaily(current, "42", "2026-08-02")
	if first.Fortune.Name != second.Fortune.Name || len(first.TodayGood) != 2 || first.TodayGood[0] == first.TodayGood[1] {
		t.Fatalf("unstable draw: %#v %#v", first, second)
	}
	if first.Fortune.Name == "吉凶未定" {
		t.Fatalf("ordinary draw selected reserved fortune: %#v", first)
	}
	for _, bad := range first.TodayBad {
		for _, good := range first.TodayGood {
			if bad == good {
				t.Fatalf("good and bad overlap: %#v", first)
			}
		}
	}
	if first.DrawSourceFingerprint == "" {
		t.Fatal("draw source fingerprint is empty")
	}
}

func TestLocalDayUsesConfiguredTimezone(t *testing.T) {
	if got := localDay(mustTime(t, "2026-08-01T18:00:00Z"), "Asia/Shanghai"); got != "2026-08-02" {
		t.Fatalf("localDay = %q", got)
	}
	if got := localDay(mustTime(t, "2026-08-01T18:00:00Z"), "UTC+08:00"); got != "2026-08-02" {
		t.Fatalf("fixed-offset localDay = %q", got)
	}
	if got := localDay(mustTime(t, "2026-08-02T01:00:00Z"), "America/Los_Angeles"); got != "2026-08-01" {
		t.Fatalf("IANA localDay = %q", got)
	}
}

func TestSpecialDateOverridesOrdinaryDraw(t *testing.T) {
	special := fortune{Name: "吉凶未定", Stars: "???????", Sign: "云深", Explanation: "静候"}
	current := settings{
		Fortunes: []fortune{{Name: "大吉", Stars: "★★★★★★★", Sign: "a", Explanation: "a"}, special},
		SpecialDates: []specialDate{
			{Date: "08-02", FortuneName: "吉凶未定"},
		},
	}
	record := drawDaily(current, "42", "2026-08-02")
	if !record.Special || record.Fortune.Name != "吉凶未定" {
		t.Fatalf("special date not applied: %#v", record)
	}
}

func TestFingerprintChangesWithDrawSettings(t *testing.T) {
	current := settings{Fortunes: []fortune{{Name: "大吉", Stars: "★★★★★★★", Sign: "a", Explanation: "a"}}}
	first := drawSourceFingerprint(current)
	current.GoodActions = []string{"整理计划"}
	if second := drawSourceFingerprint(current); first == second {
		t.Fatal("fingerprint did not change with draw settings")
	}
}

func TestMatchConfiguredCommandUsesRuntimeTriggerNames(t *testing.T) {
	current := settings{
		TriggerCommands:      []string{"我的运势", " 今日运势 "},
		StatsTriggerCommands: []string{"运势统计"},
	}
	tests := []struct {
		command string
		want    string
	}{
		{command: "我的运势", want: "fortune"},
		{command: "今日运势", want: "fortune"},
		{command: "运势统计", want: "fortune_stats"},
		{command: "fortune", want: ""},
		{command: "fortune_stats", want: ""},
		{command: "未知命令", want: ""},
	}
	for _, test := range tests {
		if got := matchConfiguredCommand(test.command, current); got != test.want {
			t.Fatalf("matchConfiguredCommand(%q) = %q, want %q", test.command, got, test.want)
		}
	}
}

func TestStatsTrackStreaksAndReplacement(t *testing.T) {
	stats := updateStats(emptyStats(), "大吉", "2026-08-01")
	stats = updateStats(stats, "大吉", "2026-08-02")
	if stats.TotalDays != 2 || stats.CurrentStreak != 2 || stats.LongestDajiStreak != 2 || stats.Counts["大吉"] != 2 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
	stats = replaceCurrentDayFortune(stats, "大吉", "大凶")
	if stats.Counts["大吉"] != 1 || stats.Counts["大凶"] != 1 || stats.CurrentDajiStreak != 1 || stats.CurrentDaxiongStreak != 1 {
		t.Fatalf("unexpected replacement stats: %#v", stats)
	}
}

func TestMergeSettingsNormalizesInvalidOverrides(t *testing.T) {
	defaults, err := defaultSettings()
	if err != nil {
		t.Fatal(err)
	}
	merged := mergeSettings(defaults, map[string]any{
		"trigger_commands": []any{" 我的运势 ", "我的运势", ""},
		"timezone":         "Mars/Olympus",
		"fortunes": []any{
			map[string]any{"name": "大吉", "stars": "错误星级", "sign": "签", "explanation": "解"},
		},
		"special_dates": []any{
			map[string]any{"date": "02-30", "fortune_name": "大吉"},
			map[string]any{"date": "02-29", "fortune_name": "大吉"},
		},
	})
	if !reflect.DeepEqual(merged.TriggerCommands, []string{"我的运势"}) {
		t.Fatalf("trigger commands = %#v", merged.TriggerCommands)
	}
	if merged.Timezone != "Asia/Shanghai" || len(merged.Fortunes) != len(defaults.Fortunes) {
		t.Fatalf("invalid overrides were not restored: %#v", merged)
	}
	if len(merged.SpecialDates) != 1 || merged.SpecialDates[0].Date != "02-29" {
		t.Fatalf("special dates = %#v", merged.SpecialDates)
	}
}

func TestRenderIdentityUsesOneBotFallbacks(t *testing.T) {
	event := &rayleabot.EventContext{Event: rayleabot.Event{
		Target: rayleabot.Target{Type: "group", ID: "100"},
		Payload: map[string]any{"onebot": map[string]any{
			"user_id": "42", "group_name": "测试群",
			"sender": map[string]any{"nickname": "昵称", "card": "群名片", "title": "头衔"},
		}},
	}}
	user := renderUser(event)
	if user["id"] != "42" || user["nickname"] != "昵称" || user["group_nickname"] != "群名片" {
		t.Fatalf("user identity = %#v", user)
	}
	if !strings.Contains(user["avatar_url"].(string), "nk=42") {
		t.Fatalf("avatar = %#v", user["avatar_url"])
	}
	if group := renderGroup(event); group["name"] != "100" {
		t.Fatalf("group fallback = %#v", group)
	}
}

func TestStatsRenderDataKeepsLocalDateAndOmitsEmptyRecords(t *testing.T) {
	stats := updateStats(emptyStats(), "大吉", "2026-08-06")
	stats = updateStats(stats, "小吉", "2026-08-07")
	stats = updateStats(stats, "小吉", "2026-08-08")
	data := statsRenderData(&rayleabot.EventContext{}, settings{Timezone: "Asia/Shanghai"}, stats)
	if data["subtitle"] == "" || len(data["summary"].([]map[string]any)) != 2 {
		t.Fatalf("stats metadata = %#v", data)
	}
	records := data["records"].([]map[string]any)
	if len(records) != 1 || records[0]["label"] != "最长连续大吉" {
		t.Fatalf("stats records = %#v", records)
	}
	distribution := data["distribution"].([]map[string]any)
	if distribution[0]["percentage"] != 33.3 {
		t.Fatalf("distribution percentage = %#v", distribution[0]["percentage"])
	}
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
