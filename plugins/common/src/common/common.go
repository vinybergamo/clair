package main

import (
	"fmt"
	"os"

	flag "github.com/spf13/pflag"
	"github.com/vinybergamo/clair/plugins/common"
)

func main() {
	quiet := flag.Bool("quiet", false, "--quiet: set CLAIR_QUIET_OUTPUT=1")
	global := flag.Bool("global", false, "--global: Whether global or app-specific")
	flag.Parse()
	cmd := flag.Arg(0)

	if *quiet {
		os.Setenv("CLAIR_QUIET_OUTPUT", "1")
	}

	var err error
	switch cmd {
	case "docker-cleanup":
		appName := flag.Arg(1)
		force := common.ToBool(flag.Arg(2))
		if *global {
			appName = "--global"
		}
		err = common.DockerCleanup(appName, force)
	case "is-deployed":
		appName := flag.Arg(1)
		if !common.IsDeployed(appName) {
			err = fmt.Errorf("App %v not deployed", appName)
		}
	case "image-is-cnb-based":
		image := flag.Arg(1)
		if common.IsImageCnbBased(image) {
			fmt.Print("true")
		} else {
			fmt.Print("false")
		}
	case "image-is-herokuish-based":
		image := flag.Arg(1)
		appName := flag.Arg(2)
		if common.IsImageHerokuishBased(image, appName) {
			fmt.Print("true")
		} else {
			fmt.Print("false")
		}
	case "scheduler-detect":
		appName := flag.Arg(1)
		if *global {
			appName = "--global"
		}
		fmt.Print(common.GetAppScheduler(appName))
	case "verify-app-name":
		appName := flag.Arg(1)
		err = common.VerifyAppName(appName)
	default:
		err = fmt.Errorf("Invalid common command call: %v", cmd)
	}

	if err != nil {
		common.LogFailWithErrorQuiet(err)
	}
}
