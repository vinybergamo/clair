package common

import (
	"fmt"
)

func TriggerAppList(filtered bool) error {
	var apps []string
	if filtered {
		apps, _ = ClairApps()
	} else {
		apps, _ = UnfilteredClairApps()
	}

	for _, app := range apps {
		Log(app)
	}

	return nil
}

func TriggerCorePostDeploy(appName string) error {
	return EnvWrap(func() error {
		CommandPropertySet("common", appName, "deployed", "true", DefaultProperties, GlobalProperties)
		return nil
	}, map[string]string{"CLAIR_QUIET_OUTPUT": "1"})
}

func TriggerInstall() error {
	if err := PropertySetup("common"); err != nil {
		return fmt.Errorf("Unable to install the common plugin: %s", err.Error())
	}

	apps, err := UnfilteredClairApps()
	if err != nil {
		return nil
	}

	for _, appName := range apps {
		IsDeployed(appName)
	}

	return nil
}

func TriggerPostAppCloneSetup(oldAppName string, newAppName string) error {
	if err := PropertyClone("common", oldAppName, newAppName); err != nil {
		return err
	}

	if err := PropertyDelete("common", oldAppName, "deployed"); err != nil {
		return err
	}

	return nil
}

func TriggerPostAppRenameSetup(oldAppName string, newAppName string) error {
	if err := PropertyClone("common", oldAppName, newAppName); err != nil {
		return err
	}

	if err := PropertyDestroy("common", oldAppName); err != nil {
		return err
	}

	return nil
}

func TriggerPostDelete(appName string) error {
	return PropertyDestroy("common", appName)
}
