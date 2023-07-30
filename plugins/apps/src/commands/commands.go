package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/vinybergamo/clair/plugins/common"
)

const (
	helpHeader = `Usage: clair apps[:COMMAND]

Manage apps

Additional commands:`

	helpContent = `
    apps:clone <old-app> <new-app>, Clones an app
    apps:create <app>, Create a new app
    apps:destroy <app>, Permanently destroy an app
    apps:exists <app>, Checks if an app exists
    apps:list, List your apps
    apps:lock <app>, Locks an app for deployment
    apps:locked <app>, Checks if an app is locked for deployment
    apps:rename <old-app> <new-app>, Rename an app
    apps:report [<app>] [<flag>], Display report about an app
    apps:unlock <app>, Unlocks an app for deployment
`
)

func main() {
	flag.Usage = usage
	flag.Parse()

	cmd := flag.Arg(0)
	switch cmd {
	case "apps", "apps:help":
		usage()
	case "help":
		command := common.NewShellCmd(fmt.Sprintf("ps -o command= %d", os.Getppid()))
		command.ShowOutput = false
		output, err := command.Output()

		if err == nil && strings.Contains(string(output), "--all") {
			fmt.Println(helpContent)
		} else {
			fmt.Print("\n    apps, Manage apps\n")
		}
	default:
		clairNotImplementExitCode, err := strconv.Atoi(os.Getenv("CLAIR_NOT_IMPLEMENTED_EXIT"))
		if err != nil {
			fmt.Println("failed to retrieve CLAIR_NOT_IMPLEMENTED_EXIT environment variable")
			clairNotImplementExitCode = 10
		}
		os.Exit(clairNotImplementExitCode)
	}
}

func usage() {
	common.CommandUsage(helpHeader, helpContent)
}
