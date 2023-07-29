package common

import (
	"fmt"
	"os"
	"strings"
)

func filterApps(apps []string) ([]string, error) {
	if !PluginTriggerExists("user-auth-app") {
		return apps, nil
	}

	sshUser := os.Getenv("SSH_USER")
	if sshUser == "" {
		sshUser = os.Getenv("USER")
	}

	sshName := os.Getenv("SSH_NAME")
	if sshName == "" {
		sshName = os.Getenv("NAME")
	}
	if sshName == "" {
		sshName = "default"
	}

	args := append([]string{sshUser, sshName}, apps...)
	b, _ := PluginTriggerOutput("user-auth-app", args...)
	filteredApps := strings.Split(strings.TrimSpace(string(b[:])), "\n")
	filteredApps = removeEmptyEntries(filteredApps)

	if len(filteredApps) == 0 {
		return filteredApps, fmt.Errorf("You haven't deployed any applications yet")
	}

	return filteredApps, nil
}

func removeEmptyEntries(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

func times(str string, n int) (out string) {
	for i := 0; i < n; i++ {
		out += str
	}
	return
}
