package apps

import (
	"fmt"
	"strings"

	"github.com/vinybergamo/clair/plugins/common"
)

func ReportSingleApp(appName string, format string, infoFlag string) error {
	if err := common.VerifyAppName(appName); err != nil {
		return err
	}

	flags := map[string]common.ReportFunc{
		"--app-created-at":             reportCreatedAt,
		"--app-deploy-source":          reportDeploySource,
		"--app-deploy-source-metadata": reportDeploySourceMetadata,
		"--app-dir":                    reportDir,
		"--app-locked":                 reportLocked,
	}

	flagKeys := []string{}
	for flagKey := range flags {
		flagKeys = append(flagKeys, flagKey)
	}

	trimPrefix := false
	uppercaseFirstCharacter := true
	infoFlags := common.CollectReport(appName, infoFlag, flags)
	return common.ReportSingleApp("app", appName, infoFlag, infoFlags, flagKeys, format, trimPrefix, uppercaseFirstCharacter)
}

func reportCreatedAt(appName string) string {
	createdAt, err := common.PropertyListGet("apps", appName, "created-at")
	if err != nil {
		return ""
	}
	return fmt.Sprint(strings.Join(createdAt, ","))
}

func reportDeploySource(appName string) string {
	return common.PropertyGet("apps", appName, "deploy-source")
}

func reportDeploySourceMetadata(appName string) string {
	return common.PropertyGet("apps", appName, "deploy-source-metadata")
}

func reportDir(appName string) string {
	return common.AppRoot(appName)
}

func reportLocked(appName string) string {
	locked := "false"
	if appIsLocked(appName) {
		locked = "true"
	}

	return locked
}
