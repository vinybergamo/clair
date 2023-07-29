package apps

import (
	"fmt"
	"os"
	"time"

	"github.com/vinybergamo/cloud/plugins/common"
)

func TriggerAppCreate(appName string) error {
	return createApp(appName)
}

func TriggerAppDestroy(appName string) error {
	return destroyApp(appName)
}

func TriggerAppExists(appName string) error {
	return appExists(appName)
}

func TriggerAppMaybeCreate(appName string) error {
	return maybeCreateApp(appName)
}

func TriggerDeploySourceSet(appName string, sourceType string, sourceMetadata string) error {
	if err := common.PropertyWrite("apps", appName, "deploy-source", sourceType); err != nil {
		return err
	}

	return common.PropertyWrite("apps", appName, "deploy-source-metadata", sourceMetadata)
}

func TriggerInstall() error {
	if err := common.PropertySetup("apps"); err != nil {
		return fmt.Errorf("Unable to install the apps plugin: %s", err.Error())
	}

	apps, err := common.UnfilteredCloudApps()
	if err != nil {
		return nil
	}

	for _, appName := range apps {
		if common.PropertyExists("apps", appName, "created-at") {
			continue
		}

		fi, err := os.Stat(common.AppRoot(appName))
		if err != nil {
			if err := common.PropertyWrite("apps", appName, "created-at", fmt.Sprintf("%d", time.Now().Unix())); err != nil {
				return err
			}

			continue
		}

		if err := common.PropertyWrite("apps", appName, "created-at", fmt.Sprintf("%d", fi.ModTime().Unix())); err != nil {
			return err
		}
	}

	return nil
}

func TriggerPostAppCloneSetup(oldAppName string, newAppName string) error {
	err := common.PropertyClone("apps", oldAppName, newAppName)
	if err != nil {
		return err
	}

	return nil
}

func TriggerPostAppRenameSetup(oldAppName string, newAppName string) error {
	if err := common.PropertyClone("apps", oldAppName, newAppName); err != nil {
		return err
	}

	if err := common.PropertyDestroy("apps", oldAppName); err != nil {
		return err
	}

	return nil
}

func TriggerPostDelete(appName string) error {
	if err := common.PropertyDestroy("apps", appName); err != nil {
		common.LogWarn(err.Error())
	}

	imagesByAppLabel, err := listImagesByAppLabel(appName)
	if err != nil {
		common.LogWarn(err.Error())
	}

	imageRepo := common.GetAppImageRepo(appName)
	imagesByRepo, err := listImagesByImageRepo(imageRepo)
	if err != nil {
		common.LogWarn(err.Error())
	}

	images := append(imagesByAppLabel, imagesByRepo...)
	common.RemoveImages(images)

	return nil
}
