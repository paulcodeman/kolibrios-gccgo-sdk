// Exercise reflection used by OpenCode's unchanged configuration decoder.
package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
	"kos"
)

type sliceError []int

func (sliceError) Error() string { return "slice error" }

func main() {
	kos.DebugString("OPENCODE_REFLECT_START")
	console, ok := kos.OpenConsole("OpenCode original reflection dependency test")
	if !ok {
		panic("console unavailable")
	}
	defer console.Close()
	checks := []struct {
		value interface{}
		want  bool
	}{
		{nil, false}, {17, true}, {"text", true},
		{(*int)(nil), true}, {make(chan int), true},
		{[]int(nil), false}, {map[string]int(nil), false},
		{(func())(nil), false}, {[2]int{1, 2}, true},
		{[2]interface{}{17, "text"}, true},
		{[2]interface{}{17, []int{1}}, false},
		{struct{ hidden interface{} }{17}, true},
		{struct{ hidden interface{} }{[]int{1}}, false},
		{struct{ E error }{sliceError{1}}, false},
		{struct{ E error }{}, true},
		{[1]struct{ E error }{{sliceError{2}}}, false},
	}
	for i, check := range checks {
		kos.DebugString("COMPARABLE_CASE:" + string(rune('A'+i)))
		if reflect.ValueOf(check.value).Comparable() != check.want {
			panic("reflection comparability")
		}
	}
	var dynamic interface{} = []int{1}
	if reflect.ValueOf(&dynamic).Elem().Comparable() {
		panic("interface comparability must inspect dynamic type")
	}
	dynamic = 23
	if !reflect.ValueOf(&dynamic).Elem().Comparable() {
		panic("comparable dynamic interface")
	}
	var e error = sliceError{7}
	ev := reflect.ValueOf(&e).Elem().Elem()
	if ev.Kind() != reflect.Slice || ev.Len() != 1 || ev.Index(0).Int() != 7 {
		panic("nonempty interface dynamic value")
	}
	private := reflect.ValueOf(struct{ hidden interface{} }{17}).Field(0).Elem()
	if private.CanInterface() {
		panic("interface Elem must preserve unexported field restriction")
	}
	kos.DebugString("REFLECT_COMPARABLE_INTERFACES_OK")
	var config struct {
		Name    string
		Retries int
		Enabled bool
	}
	err := mapstructure.Decode(map[string]interface{}{
		"Name": "KolibriOS", "Retries": 3, "Enabled": true,
	}, &config)
	if err != nil || config.Name != "KolibriOS" || config.Retries != 3 || !config.Enabled {
		panic("unchanged configuration decoder")
	}
	kos.DebugString("ORIGINAL_MAPSTRUCTURE_OK")
	if err := os.Setenv("OPENCODE_TEST_EXPAND", "KolibriOS"); err != nil {
		panic(err)
	}
	defer os.Unsetenv("OPENCODE_TEST_EXPAND")
	if os.ExpandEnv("$OPENCODE_TEST_EXPAND/${OPENCODE_TEST_EXPAND}") != "KolibriOS/KolibriOS" {
		panic("upstream environment expansion")
	}
	for _, check := range []struct{ input, want string }{
		{"plain", "plain"}, {"$", "$"}, {"$?", "[?]"},
		{"${}", ""}, {"${unfinished", "unfinished"},
		{"$name/$1/$*", "[name]/[1]/[*]"}, {"$-/$#/$@", "[-]/[#]/[@]"},
		{"$!/$9/$0/$$", "[!]/[9]/[0]/[$]"}, {"$a_9!", "[a_9]!"},
		{"${ spaced name }", "[ spaced name ]"},
	} {
		if os.Expand(check.input, func(key string) string { return "[" + key + "]" }) != check.want {
			panic("upstream shell-name parsing")
		}
	}
	settings := viper.New()
	settings.SetConfigType("json")
	if err := settings.ReadConfig(strings.NewReader(`{"name":"original","retries":4,"enabled":true}`)); err != nil {
		panic(err)
	}
	if err := settings.BindEnv("name", "OPENCODE_TEST_EXPAND"); err != nil {
		panic(err)
	}
	if settings.GetString("name") != "KolibriOS" || settings.GetInt("retries") != 4 || !settings.GetBool("enabled") {
		panic("original Viper configuration and environment override")
	}
	const root = "/hd0/1/VIPCASE"
	if err := os.RemoveAll(root); err != nil {
		panic(err)
	}
	defer os.RemoveAll(root)
	if err := os.MkdirAll(root+"/sub", 0700); err != nil {
		panic(err)
	}
	if err := os.WriteFile(root+"/settings.json", []byte(`{"name":"file","retries":6}`), 0600); err != nil {
		panic(err)
	}
	if path, err := filepath.EvalSymlinks(root + "/sub/../settings.json"); err != nil || path != root+"/settings.json" {
		panic("original path evaluation with parent component")
	}
	if _, err := filepath.EvalSymlinks(root + "/missing"); !errors.Is(err, fs.ErrNotExist) {
		panic("path evaluation missing component")
	}
	if _, err := filepath.EvalSymlinks(root + "/settings.json/child"); !errors.Is(err, syscall.ENOTDIR) {
		panic("path evaluation non-directory component")
	}
	settings.SetConfigFile(root + "/settings.json")
	if err := settings.ReadInConfig(); err != nil || settings.GetInt("retries") != 6 {
		panic("original Viper file loading")
	}
	kos.DebugString("ORIGINAL_PATH_EVALUATION_OK")
	for _, key := range []string{"HOME", "XDG_CACHE_HOME", "XDG_CONFIG_HOME"} {
		if err := os.Unsetenv(key); err != nil {
			panic(err)
		}
	}
	if home, err := os.UserHomeDir(); err == nil || home != "" {
		panic("missing home directory must report an error")
	}
	if _, err := os.UserCacheDir(); err == nil {
		panic("missing user cache directory must report an error")
	}
	if _, err := os.UserConfigDir(); err == nil {
		panic("missing user configuration directory must report an error")
	}
	if err := os.Setenv("HOME", root); err != nil {
		panic(err)
	}
	if home, err := os.UserHomeDir(); err != nil || home != root {
		panic("original home directory lookup")
	}
	if cache, err := os.UserCacheDir(); err != nil || cache != root+"/.cache" {
		panic("original default cache directory")
	}
	if config, err := os.UserConfigDir(); err != nil || config != root+"/.config" {
		panic("original default configuration directory")
	}
	if err := os.Setenv("XDG_CACHE_HOME", root+"/cache"); err != nil {
		panic(err)
	}
	if err := os.Setenv("XDG_CONFIG_HOME", root+"/config"); err != nil {
		panic(err)
	}
	if cache, err := os.UserCacheDir(); err != nil || cache != root+"/cache" {
		panic("original cache directory override")
	}
	if config, err := os.UserConfigDir(); err != nil || config != root+"/config" {
		panic("original configuration directory override")
	}
	kos.DebugString("ORIGINAL_USER_DIRECTORIES_OK")
	kos.DebugString("ORIGINAL_VIPER_ENV_OK")
	kos.DebugString("OPENCODE_REFLECT_PASS")
	console.WriteString("OPENCODE_REFLECT_PASS\n")
}
