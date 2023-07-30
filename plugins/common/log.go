package common

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

type ErrWithExitCode interface {
	ExitCode() int
}

type writer struct {
	mu     *sync.Mutex
	source string
}

func (w *writer) Write(bytes []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.source == "stdout" {
		for _, line := range strings.Split(string(bytes), "\n") {
			if line == "" {
				continue
			}

			LogVerboseQuiet(line)
		}
	} else {
		for _, line := range strings.Split(string(bytes), "\n") {
			if line == "" {
				continue
			}

			LogVerboseStderrQuiet(line)
		}
	}

	return len(bytes), nil
}

func LogFail(text string) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(" !     %s", text))
	os.Exit(1)
}

func LogFailWithError(err error) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(" !     %s", err.Error()))
	if errExit, ok := err.(ErrWithExitCode); ok {
		os.Exit(errExit.ExitCode())
	}
	os.Exit(1)
}

func LogFailWithErrorQuiet(err error) {
	if os.Getenv("CLAIR_QUIET_OUTPUT") == "" {
		fmt.Fprintln(os.Stderr, fmt.Sprintf(" !     %s", err.Error()))
	}
	if errExit, ok := err.(ErrWithExitCode); ok {
		os.Exit(errExit.ExitCode())
	}
	os.Exit(1)
}

func LogFailQuiet(text string) {
	if os.Getenv("CLAIR_QUIET_OUTPUT") == "" {
		fmt.Fprintln(os.Stderr, fmt.Sprintf(" !     %s", text))
	}
	os.Exit(1)
}

func Log(text string) {
	fmt.Println(text)
}

func LogQuiet(text string) {
	if os.Getenv("CLAIR_QUIET_OUTPUT") == "" {
		fmt.Println(text)
	}
}

func LogInfo1(text string) {
	fmt.Println(fmt.Sprintf("-----> %s", text))
}

func LogInfo1Quiet(text string) {
	if os.Getenv("CLAIR_QUIET_OUTPUT") == "" {
		LogInfo1(text)
	}
}

func LogInfo2(text string) {
	fmt.Println(fmt.Sprintf("=====> %s", text))
}

func LogInfo2Quiet(text string) {
	if os.Getenv("CLAIR_QUIET_OUTPUT") == "" {
		LogInfo2(text)
	}
}

func LogVerbose(text string) {
	fmt.Println(fmt.Sprintf("       %s", text))
}

func LogVerboseStderr(text string) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(" !     %s", text))
}

func LogVerboseQuiet(text string) {
	if os.Getenv("CLAIR_QUIET_OUTPUT") == "" {
		LogVerbose(text)
	}
}

func LogVerboseStderrQuiet(text string) {
	if os.Getenv("CLAIR_QUIET_OUTPUT") == "" {
		LogVerboseStderr(text)
	}
}

func LogVerboseQuietContainerLogs(containerID string) {
	LogVerboseQuietContainerLogsTail(containerID, 0, false)
}

func LogVerboseQuietContainerLogsTail(containerID string, lines int, tail bool) {
	args := []string{"container", "logs", containerID}
	if lines > 0 {
		args = append(args, "--tail", strconv.Itoa(lines))
	}
	if tail {
		args = append(args, "--follow")
	}

	sc := NewShellCmdWithArgs(DockerBin(), args...)
	var mu sync.Mutex
	sc.Command.Stdout = &writer{
		mu:     &mu,
		source: "stdout",
	}
	sc.Command.Stderr = &writer{
		mu:     &mu,
		source: "stderr",
	}

	if err := sc.Command.Start(); err != nil {
		LogExclaim(fmt.Sprintf("Failed to fetch container logs: %s", containerID))
	}

	if err := sc.Command.Wait(); err != nil {
		LogExclaim(fmt.Sprintf("Failed to fetch container logs: %s", containerID))
	}
}

// LogWarn is the warning log formatter
func LogWarn(text string) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(" !     %s", text))
}

func LogExclaim(text string) {
	fmt.Println(fmt.Sprintf(" !     %s", text))
}

func LogStderr(text string) {
	fmt.Fprintln(os.Stderr, text)
}

func LogDebug(text string) {
	if os.Getenv("CLAIR_TRACE") == "1" {
		fmt.Fprintln(os.Stderr, fmt.Sprintf(" ?     %s", strings.TrimPrefix(text, " ?     ")))
	}
}
