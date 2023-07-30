package common

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/otiai10/copy"
)

func CreateAppDataDirectory(pluginName, appName string) error {
	directory := GetAppDataDirectory(pluginName, appName)
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}

	if err := SetPermissions(directory, 0755); err != nil {
		return err
	}

	return nil
}

func CreateDataDirectory(pluginName string) error {
	directory := GetDataDirectory(pluginName)
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}

	if err := SetPermissions(directory, 0755); err != nil {
		return err
	}

	return nil
}

func GetAppDataDirectory(pluginName string, appName string) string {
	return filepath.Join(GetDataDirectory(pluginName), appName)
}

func GetDataDirectory(pluginName string) string {
	return filepath.Join(MustGetEnv("CLAIR_LIB_ROOT"), "data", pluginName)
}

func MigrateAppDataDirectory(pluginName string, oldAppName string, newAppName string) error {
	if err := CloneAppData(pluginName, oldAppName, newAppName); err != nil {
		return err
	}

	return RemoveAppDataDirectory(pluginName, oldAppName)
}

func RemoveAppDataDirectory(pluginName, appName string) error {
	return os.RemoveAll(GetAppDataDirectory(pluginName, appName))
}

func CloneAppData(pluginName string, oldAppName string, newAppName string) error {
	oldDataDir := GetAppDataDirectory(pluginName, oldAppName)
	if !DirectoryExists(oldDataDir) {
		return CreateAppDataDirectory(pluginName, newAppName)
	}

	newDataDir := GetAppDataDirectory(pluginName, newAppName)
	if err := copy.Copy(oldDataDir, newDataDir); err != nil {
		return fmt.Errorf("Unable to clone app data to new location: %v", err.Error())
	}

	return nil
}

func SetupAppData(pluginName string) error {
	if err := CreateDataDirectory(pluginName); err != nil {
		return err
	}

	apps, err := UnfilteredClairApps()
	if err != nil {
		return nil
	}

	for _, appName := range apps {
		if err := CreateAppDataDirectory(pluginName, appName); err != nil {
			return err
		}
	}

	return nil
}
