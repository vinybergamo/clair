package apps

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/vinybergamo/cloud/plugins/common"
)

func appExists(appName string) error {
	return common.VerifyAppName(appName)
}

func appIsLocked(appName string) bool {
	lockfilePath := fmt.Sprintf("%v/.deploy.lock", common.AppRoot(appName))
	_, err := os.Stat(lockfilePath)
	return !os.IsNotExist(err)
}

func createApp(appName string) error {
	if err := common.IsValidAppName(appName); err != nil {
		return err
	}

	if err := appExists(appName); err == nil {
		return errors.New("Name is already taken")
	}

	common.LogInfo1Quiet(fmt.Sprintf("Creating %s...", appName))
	os.MkdirAll(common.AppRoot(appName), 0755)

	if err := common.PropertyWrite("apps", appName, "created-at", fmt.Sprintf("%d", time.Now().Unix())); err != nil {
		return err
	}

	if err := common.PluginTrigger("post-create", []string{appName}...); err != nil {
		return err
	}

	return nil
}

func destroyApp(appName string) error {
	if os.Getenv("CLOUD_APPS_FORCE_DELETE") != "1" {
		if err := common.AskForDestructiveConfirmation(appName, "app"); err != nil {
			return err
		}
	}

	common.LogInfo1(fmt.Sprintf("Destroying %s (including all add-ons)", appName))

	imageTag, _ := common.GetRunningImageTag(appName, "")
	if err := common.PluginTrigger("pre-delete", []string{appName, imageTag}...); err != nil {
		return err
	}

	scheduler := common.GetAppScheduler(appName)
	removeContainers := "true"
	if err := common.PluginTrigger("scheduler-stop", []string{scheduler, appName, removeContainers}...); err != nil {
		return err
	}
	if err := common.PluginTrigger("scheduler-post-delete", []string{scheduler, appName, imageTag}...); err != nil {
		return err
	}
	if err := common.PluginTrigger("post-delete", []string{appName, imageTag}...); err != nil {
		return err
	}

	forceCleanup := true
	common.DockerCleanup(appName, forceCleanup)

	common.LogInfo1("Retiring old containers and images")
	if err := common.PluginTrigger("scheduler-retire", []string{scheduler, appName}...); err != nil {
		return err
	}

	if err := os.RemoveAll(fmt.Sprintf("%v/", common.AppRoot(appName))); err != nil {
		common.LogWarn(err.Error())
	}

	if err := os.RemoveAll(common.AppRoot(appName)); err != nil {
		common.LogWarn(err.Error())
	}

	return nil
}

func listImagesByAppLabel(appName string) ([]string, error) {
	command := []string{
		common.DockerBin(),
		"image",
		"list",
		"--quiet",
		"--filter",
		fmt.Sprintf("label=com.cloud.app-name=%v", appName),
	}

	var stderr bytes.Buffer
	listCmd := common.NewShellCmd(strings.Join(command, " "))
	listCmd.ShowOutput = false
	listCmd.Command.Stderr = &stderr
	b, err := listCmd.Output()

	if err != nil {
		return []string{}, errors.New(strings.TrimSpace(stderr.String()))
	}

	output := strings.Split(strings.TrimSpace(string(b[:])), "\n")
	return output, nil
}

func listImagesByImageRepo(imageRepo string) ([]string, error) {
	command := []string{
		common.DockerBin(),
		"image",
		"list",
		"--quiet",
		imageRepo,
	}

	var stderr bytes.Buffer
	listCmd := common.NewShellCmd(strings.Join(command, " "))
	listCmd.ShowOutput = false
	listCmd.Command.Stderr = &stderr
	b, err := listCmd.Output()

	if err != nil {
		return []string{}, errors.New(strings.TrimSpace(stderr.String()))
	}

	output := strings.Split(strings.TrimSpace(string(b[:])), "\n")
	return output, nil
}

func maybeCreateApp(appName string) error {
	if err := appExists(appName); err == nil {
		return nil
	}

	b, _ := common.PluginTriggerOutput("config-get-global", []string{"CLOUD_DISABLE_APP_AUTOCREATION"}...)
	disableAutocreate := strings.TrimSpace(string(b[:]))
	if disableAutocreate == "true" {
		common.LogWarn("App auto-creation disabled.")
		return fmt.Errorf("Re-enable app auto-creation or create an app with 'cloud apps:create %s'", appName)
	}

	return common.SuppressOutput(func() error {
		return createApp(appName)
	})
}
