package common

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/codeskyblue/go-sh"
)

func ContainerIsRunning(containerID string) bool {
	b, err := DockerInspect(containerID, "'{{.State.Running}}'")
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(b[:])) == "true"
}

func ContainerStart(containerID string) bool {
	cmd := sh.Command(DockerBin(), "container", "start", containerID)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return false
	}

	return true
}

func ContainerExists(containerID string) bool {
	cmd := sh.Command(DockerBin(), "container", "inspect", containerID)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return false
	}

	return true
}

func ContainerWaitTilReady(containerID string, timeout time.Duration) error {
	time.Sleep(timeout)

	if !ContainerIsRunning(containerID) {
		return fmt.Errorf("Container %s is not running", containerID)
	}

	return nil
}

func CopyFromImage(appName string, image string, source string, destination string) error {
	if !VerifyImage(image) {
		return fmt.Errorf("Invalid docker image for copying content")
	}

	workDir := ""
	if !IsAbsPath(source) {
		if IsImageCnbBased(image) {
			workDir = "/workspace"
		} else if IsImageHerokuishBased(image, appName) {
			workDir = "/app"
		} else {
			workDir, _ = DockerInspect(image, "{{.Config.WorkingDir}}")
		}

		if workDir != "" {
			source = fmt.Sprintf("%s/%s", workDir, source)
		}
	}

	tmpFile, err := ioutil.TempFile(os.TempDir(), fmt.Sprintf("cloud-%s-%s", MustGetEnv("CLOUD_PID"), "CopyFromImage"))
	if err != nil {
		return fmt.Errorf("Cannot create temporary file: %v", err)
	}

	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	globalRunArgs := MustGetEnv("CLOUD_GLOBAL_RUN_ARGS")
	createLabelArgs := []string{"--label", fmt.Sprintf("com.cloud.app-name=%s", appName), globalRunArgs}
	containerID, err := DockerContainerCreate(image, createLabelArgs)
	if err != nil {
		return fmt.Errorf("Unable to create temporary container: %v", err)
	}

	containerCopyCmd := NewShellCmd(strings.Join([]string{
		DockerBin(),
		"container",
		"cp",
		fmt.Sprintf("%s:%s", containerID, source),
		tmpFile.Name(),
	}, " "))
	containerCopyCmd.ShowOutput = false
	fileCopied := containerCopyCmd.Execute()

	containerRemoveCmd := NewShellCmd(strings.Join([]string{
		DockerBin(),
		"container",
		"rm",
		"--force",
		containerID,
	}, " "))
	containerRemoveCmd.ShowOutput = false
	containerRemoveCmd.Execute()

	if !fileCopied {
		return fmt.Errorf("Unable to copy file %s from image", source)
	}

	fi, err := os.Stat(tmpFile.Name())
	if err != nil {
		return err
	}

	if fi.Size() == 0 {
		return fmt.Errorf("Unable to copy file %s from image", source)
	}

	dos2unixCmd := NewShellCmd(strings.Join([]string{
		"dos2unix",
		"-l",
		"-n",
		tmpFile.Name(),
		destination,
	}, " "))
	dos2unixCmd.ShowOutput = false
	dos2unixCmd.Execute()

	b, err := sh.Command("tail", "-c1", destination).Output()
	if string(b) != "" {
		f, err := os.OpenFile(destination, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := f.WriteString("\n"); err != nil {
			return fmt.Errorf("Unable to append trailing newline to copied file: %v", err)
		}
	}

	return nil
}

func DockerBin() string {
	dockerBin := os.Getenv("DOCKER_BIN")
	if dockerBin == "" {
		dockerBin = "docker"
	}

	return dockerBin
}

func DockerCleanup(appName string, forceCleanup bool) error {
	if !forceCleanup {
		skipCleanup := false
		if appName != "" {
			triggerName := "config-get"
			triggerArgs := []string{appName, "CLOUD_SKIP_CLEANUP"}
			if appName == "--global" {
				triggerName = "config-get-global"
				triggerArgs = []string{"CLOUD_SKIP_CLEANUP"}
			}

			b, _ := PluginTriggerOutput(triggerName, triggerArgs...)
			output := strings.TrimSpace(string(b[:]))
			if output == "true" {
				skipCleanup = true
			}
		}

		if skipCleanup || os.Getenv("CLOUD_SKIP_CLEANUP") == "true" {
			LogInfo1("CLOUD_SKIP_CLEANUP set. Skipping cloud cleanup")
			return nil
		}
	}

	LogInfo1("Cleaning up...")
	if appName == "--global" {
		appName = ""
	}

	exitedContainerIDs, _ := listContainers("exited", appName)
	deadContainerIDs, _ := listContainers("dead", appName)
	containerIDs := append(exitedContainerIDs, deadContainerIDs...)

	if len(containerIDs) > 0 {
		removeContainers(containerIDs)
	}

	imageIDs, _ := ListDanglingImages(appName)
	if len(imageIDs) > 0 {
		RemoveImages(imageIDs)
	}

	if appName != "" {
		// delete unused images
		pruneUnusedImages(appName)
	}

	return nil
}

func DockerContainerCreate(image string, containerCreateArgs []string) (string, error) {
	cmd := []string{
		DockerBin(),
		"container",
		"create",
	}

	cmd = append(cmd, containerCreateArgs...)
	cmd = append(cmd, image)

	var stderr bytes.Buffer
	containerCreateCmd := NewShellCmd(strings.Join(cmd, " "))
	containerCreateCmd.ShowOutput = false
	containerCreateCmd.Command.Stderr = &stderr
	b, err := containerCreateCmd.Output()
	if err != nil {
		return "", errors.New(strings.TrimSpace(stderr.String()))
	}

	return strings.TrimSpace(string(b[:])), nil
}

func DockerInspect(containerOrImageID, format string) (output string, err error) {
	b, err := sh.Command(DockerBin(), "inspect", "--format", format, containerOrImageID).Output()
	if err != nil {
		return "", err
	}
	output = strings.TrimSpace(string(b[:]))
	if strings.HasPrefix(output, "'") && strings.HasSuffix(output, "'") {
		output = strings.TrimSuffix(strings.TrimPrefix(output, "'"), "'")
	}
	return
}

func IsImageCnbBased(image string) bool {
	if len(image) == 0 {
		return false
	}

	output, err := DockerInspect(image, "{{index .Config.Labels \"io.buildpacks.stack.id\" }}")
	if err != nil {
		return false
	}
	return output != ""
}

func IsImageHerokuishBased(image string, appName string) bool {
	if len(image) == 0 {
		return false
	}

	if IsImageCnbBased(image) {
		return true
	}

	cloudAppUser := ""
	if len(appName) != 0 {
		b, err := PluginTriggerOutput("config-get", []string{appName, "CLOUD_APP_USER"}...)
		if err == nil {
			cloudAppUser = strings.TrimSpace(string(b))
		}
	}

	if len(cloudAppUser) == 0 {
		cloudAppUser = "herokuishuser"
	}

	output, err := DockerInspect(image, fmt.Sprintf("{{range .Config.Env}}{{if eq . \"USER=%s\" }}{{println .}}{{end}}{{end}}", cloudAppUser))
	if err != nil {
		return false
	}
	return output != ""
}

func ListDanglingImages(appName string) ([]string, error) {
	command := []string{
		DockerBin(),
		"image",
		"list",
		"--quiet",
		"--filter",
		"dangling=true",
	}

	if appName != "" {
		command = append(command, []string{"--filter", fmt.Sprintf("label=com.cloud.app-name=%v", appName)}...)
	}

	var stderr bytes.Buffer
	listCmd := NewShellCmd(strings.Join(command, " "))
	listCmd.ShowOutput = false
	listCmd.Command.Stderr = &stderr
	b, err := listCmd.Output()

	if err != nil {
		return []string{}, errors.New(strings.TrimSpace(stderr.String()))
	}

	output := strings.Split(strings.TrimSpace(string(b[:])), "\n")
	return output, nil
}

func RemoveImages(imageIDs []string) {
	command := []string{
		DockerBin(),
		"image",
		"rm",
	}

	command = append(command, imageIDs...)

	var stderr bytes.Buffer
	rmCmd := NewShellCmd(strings.Join(command, " "))
	rmCmd.ShowOutput = false
	rmCmd.Command.Stderr = &stderr
	rmCmd.Execute()
}

func VerifyImage(image string) bool {
	imageCmd := NewShellCmd(strings.Join([]string{DockerBin(), "image", "inspect", image}, " "))
	imageCmd.ShowOutput = false
	return imageCmd.Execute()
}

func listContainers(status string, appName string) ([]string, error) {
	command := []string{
		DockerBin(),
		"container",
		"list",
		"--quiet",
		"--all",
		"--filter",
		fmt.Sprintf("status=%v", status),
		"--filter",
		fmt.Sprintf("label=%v", os.Getenv("CLOUD_CONTAINER_LABEL")),
	}

	if appName != "" {
		command = append(command, []string{"--filter", fmt.Sprintf("label=com.cloud.app-name=%v", appName)}...)
	}

	var stderr bytes.Buffer
	listCmd := NewShellCmd(strings.Join(command, " "))
	listCmd.ShowOutput = false
	listCmd.Command.Stderr = &stderr
	b, err := listCmd.Output()

	if err != nil {
		return []string{}, errors.New(strings.TrimSpace(stderr.String()))
	}

	output := strings.Split(strings.TrimSpace(string(b[:])), "\n")
	return output, nil
}

func pruneUnusedImages(appName string) {
	command := []string{
		DockerBin(),
		"image",
		"prune",
		"--all",
		"--force",
		"--filter",
		fmt.Sprintf("label=com.cloud.app-name=%v", appName),
	}

	var stderr bytes.Buffer
	pruneCmd := NewShellCmd(strings.Join(command, " "))
	pruneCmd.ShowOutput = false
	pruneCmd.Command.Stderr = &stderr
	pruneCmd.Execute()
}

func removeContainers(containerIDs []string) {
	command := []string{
		DockerBin(),
		"container",
		"rm",
	}

	command = append(command, containerIDs...)

	var stderr bytes.Buffer
	rmCmd := NewShellCmd(strings.Join(command, " "))
	rmCmd.ShowOutput = false
	rmCmd.Command.Stderr = &stderr
	rmCmd.Execute()
}
