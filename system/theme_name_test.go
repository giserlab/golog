package system

import (
	"testing"

	"golog/entity"
)

// useLocale 临时把全局配置切到指定语言，测试结束后还原。
func useLocale(t *testing.T, locale string) {
	t.Helper()
	original := Config
	Config = &entity.Config{Theme: "default", Locale: locale}
	t.Cleanup(func() { Config = original })
}

// TestThemeNameLocalized 主题展示名称必须跟随界面语言变化，名称由各主题
// 自己的 locales/theme_name 提供。
func TestThemeNameLocalized(t *testing.T) {
	cases := []struct {
		locale string
		want   map[string]string
	}{
		{"zh-cn", map[string]string{"default": "默认", "note": "笔记", "corporate": "企业"}},
		{"zh-tw", map[string]string{"default": "預設", "note": "笔记", "corporate": "企業"}},
		{"en-us", map[string]string{"default": "Default", "note": "Note", "corporate": "Corporate"}},
	}

	for _, tc := range cases {
		t.Run(tc.locale, func(t *testing.T) {
			useLocale(t, tc.locale)
			for theme, want := range tc.want {
				if got := ThemeName(theme); got != want {
					t.Errorf("ThemeName(%q) locale=%s = %q, want %q", theme, tc.locale, got, want)
				}
			}
		})
	}
}

// TestThemeNameFallback 主题未提供翻译或主题不存在时，必须回退到
// default.json 的名称或目录名，保证下拉框永远有可读文本。
func TestThemeNameFallback(t *testing.T) {
	// 主题没有该语言文件时，回退到基准语言（default.json）。
	useLocale(t, "fr-fr")
	if got := ThemeName("note"); got != "Note" {
		t.Errorf("expected fallback to the base locale, got %q", got)
	}

	// 完全没有 locales 的主题回退为目录名。
	if got := ThemeName("my-custom-theme"); got != "my-custom-theme" {
		t.Errorf("unknown theme should fall back to its directory name, got %q", got)
	}

	if got := ThemeName(""); got != "" {
		t.Errorf("empty theme name should stay empty, got %q", got)
	}
}

// TestThemeInfosCoverEveryTheme 下拉框数据必须与 Themes() 一一对应：
// Value 是目录名（提交值），Name 是本地化名称且不能等于目录名。
func TestThemeInfosCoverEveryTheme(t *testing.T) {
	useLocale(t, "zh-cn")

	names := Themes()
	infos := ThemeInfos()
	if len(infos) != len(names) {
		t.Fatalf("ThemeInfos() returned %d entries, Themes() returned %d", len(infos), len(names))
	}
	for i, info := range infos {
		if info.Value != names[i] {
			t.Errorf("entry %d value = %q, want %q", i, info.Value, names[i])
		}
		if info.Name == "" {
			t.Errorf("theme %q has an empty display name", info.Value)
		}
		if info.Name == info.Value {
			t.Errorf("theme %q is missing a localized theme_name in its locales", info.Value)
		}
	}
}
