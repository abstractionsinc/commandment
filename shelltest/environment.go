package shelltest

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

const frameworkEnvironmentCapacityReserve = 16

func frameworkEnvironment(workspace string) ([]string, error) {
	home := filepath.Join(workspace, "home")
	config := filepath.Join(workspace, "xdg-config")
	cache := filepath.Join(workspace, "xdg-cache")
	data := filepath.Join(workspace, "xdg-data")
	state := filepath.Join(workspace, "xdg-state")
	zdotdir := filepath.Join(workspace, "zdotdir")
	inputrc := filepath.Join(workspace, "inputrc")
	histfile := filepath.Join(workspace, "history")
	emptyStartup := filepath.Join(workspace, "empty-startup")

	for _, directory := range []string{home, config, cache, data, state, zdotdir} {
		if err := os.MkdirAll(directory, privateDirectoryMode); err != nil {
			failure := &failureError{
				subject: "create deterministic shell directory",
				detail:  directory,
				cause:   err,
			}
			return nil, failure
		}
	}
	for _, file := range []string{inputrc, histfile, emptyStartup} {
		if err := os.WriteFile(file, nil, privateFileMode); err != nil {
			failure := &failureError{
				subject: "create deterministic shell startup file",
				detail:  file,
				cause:   err,
			}
			return nil, failure
		}
	}

	base := make([]string, 0, len(os.Environ())+frameworkEnvironmentCapacityReserve)
	for _, entry := range os.Environ() {
		name, _, ok := strings.Cut(entry, "=")
		if !ok || name == "" {
			continue
		}
		if strings.HasPrefix(name, "BASH_FUNC_") {
			continue
		}
		switch environmentKey(name) {
		case environmentKey("ENV"), environmentKey("BASH_ENV"), environmentKey("HOME"), environmentKey("XDG_CONFIG_HOME"), environmentKey("XDG_CACHE_HOME"), environmentKey("XDG_DATA_HOME"), environmentKey("XDG_STATE_HOME"), environmentKey("ZDOTDIR"), environmentKey("INPUTRC"), environmentKey("HISTFILE"), environmentKey("PROMPT_COMMAND"), environmentKey("PS1"), environmentKey("PS2"), environmentKey("PS3"), environmentKey("PS4"):
			continue
		}
		base = append(base, entry)
	}

	overrides := []string{
		"HOME=" + home,
		"XDG_CONFIG_HOME=" + config,
		"XDG_CACHE_HOME=" + cache,
		"XDG_DATA_HOME=" + data,
		"XDG_STATE_HOME=" + state,
		"ZDOTDIR=" + zdotdir,
		"INPUTRC=" + inputrc,
		"HISTFILE=" + histfile,
		"ENV=" + emptyStartup,
		"BASH_ENV=" + emptyStartup,
		"TERM=xterm-256color",
	}
	if runtime.GOOS == windowsOS {
		overrides = append(overrides,
			"USERPROFILE="+home,
			"APPDATA="+config,
			"LOCALAPPDATA="+data,
		)
	}
	return overlayEnvironment(base, overrides...), nil
}

func shellEnvironment(workspace, helperDirectory string, plan setupPlan) ([]string, error) {
	base, err := frameworkEnvironment(workspace)
	if err != nil {
		return nil, err
	}
	for _, entry := range plan.environment {
		base = overlayEnvironment(base, entry.name+"="+entry.value)
	}
	pathValue, _ := environmentValue(base, "PATH")
	pathEntry := helperDirectory
	if pathValue != "" {
		pathEntry += string(os.PathListSeparator) + pathValue
	}
	base = overlayEnvironment(base, "PATH="+pathEntry)
	return base, nil
}

func overlayEnvironment(base []string, overrides ...string) []string {
	values := make(map[string]string, len(base)+len(overrides))
	order := make([]string, 0, len(base)+len(overrides))
	add := func(entry string) {
		name, value, ok := strings.Cut(entry, "=")
		if !ok || name == "" {
			return
		}
		key := environmentKey(name)
		if _, exists := values[key]; !exists {
			order = append(order, key)
		}
		values[key] = name + "=" + value
	}
	for _, entry := range base {
		add(entry)
	}
	for _, entry := range overrides {
		add(entry)
	}
	result := make([]string, 0, len(order))
	for _, key := range order {
		result = append(result, values[key])
	}
	return result
}

func environmentValue(environment []string, name string) (string, bool) {
	key := environmentKey(name)
	for _, e := range slices.Backward(environment) {
		entryName, value, ok := strings.Cut(e, "=")
		if ok && environmentKey(entryName) == key {
			return value, true
		}
	}
	return "", false
}

func validEnvironmentName(name string) bool {
	return name != "" && !strings.ContainsAny(name, "=\x00")
}

func environmentKey(name string) string {
	if runtime.GOOS == windowsOS {
		return strings.ToUpper(name)
	}
	return name
}
