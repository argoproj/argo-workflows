package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"applatix.io/api"
	"applatix.io/axamm/application"
	"applatix.io/axamm/deployment"
	"applatix.io/axerror"
	humanize "github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

var (
	appListArgs appListFlags
	appShowArgs appShowFlags
)

type appListFlags struct {
	showAll bool
}

type appShowFlags struct {
}

func init() {
	RootCmd.AddCommand(appCmd)

	appCmd.AddCommand(appListCmd)
	appListCmd.Flags().BoolVar(&appListArgs.showAll, "show-all", false, "Show all applications, including terminated")

	appCmd.AddCommand(appShowCmd)
}

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "application commands",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}

var appListCmd = &cobra.Command{
	Use:   "list",
	Short: "List applications",
	Run:   appList,
}

var appShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display details about an application",
	Run:   appShow,
}

func appList(cmd *cobra.Command, args []string) {
	params := api.ApplicationListParams{}
	if !appListArgs.showAll {
		params.Statuses = api.RunningApplicationStates
	}
	initClient()
	apps, axErr := apiClient.ApplicationList(params)
	checkFatal(axErr)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tAGE")
	for _, app := range apps {
		cTime := time.Unix(app.Ctime, 0)
		now := time.Now()
		hrTimeDiff := humanize.RelTime(cTime, now, "", "later")
		fmt.Fprintf(w, "%s\t%s\t%s\n", app.Name, app.Status, hrTimeDiff)
	}
	w.Flush()
}

func appShow(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		cmd.HelpFunc()(cmd, args)
		os.Exit(1)
	}
	initClient()
	app, axErr := apiClient.ApplicationGetByName(args[0])
	checkFatal(axErr)
	if app == nil {
		axErr = axerror.ERR_AX_ILLEGAL_ARGUMENT.NewWithMessagef("Application with name '%s' not found")
		checkFatal(axErr)
	}
	printApp(app)
}

// appURL returns the app URL given an app ID
func appURL(id string) string {
	return fmt.Sprintf("%s/app/applications/details/%s", apiClient.Config.URL, id)
}

func formatEndpoint(ep string) string {
	return fmt.Sprintf("https://%s", strings.TrimRight(ep, "."))
}

func printApp(app *application.Application) {
	const fmtStr = "%-13s%v\n"
	fmt.Printf(fmtStr, "ID:", app.ID)
	fmt.Printf(fmtStr, "URL:", appURL(app.ID))
	fmt.Printf(fmtStr, "Name:", app.Name)
	fmt.Printf(fmtStr, "Status:", app.Status)
	for key, valIf := range app.StatusDetail {
		if valIf == nil {
			continue
		}
		if val, ok := valIf.(string); ok && val != "" && strings.ToLower(val) != strings.ToLower(app.Status) {
			fmt.Printf(fmtStr, "  "+key+":", val)
		}
	}
	if len(app.Endpoints) > 1 {
		fmt.Printf(fmtStr, "Endpoints:", "")
		for _, ep := range app.Endpoints {
			fmt.Println("- " + formatEndpoint(ep))
		}
	}

	fmt.Println()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "%s\n", app.Name)
	fmt.Fprintf(w, " |\n")
	for i, dep := range app.Deployments {
		isLast := i == len(app.Deployments)-1
		var prefix string
		if isLast {
			prefix = " └"
		} else {
			prefix = " ├"
		}
		fmt.Fprintf(w, "%s- %s\t%s\t%s\n", prefix, dep.Name, dep.Status, dep.Id)
		printDeployment(w, dep, isLast)
	}
	w.Flush()
}

func printDeployment(w *tabwriter.Writer, dep *deployment.Deployment, isLastDep bool) {
	var prefix string
	if isLastDep {
		prefix = "     "
	} else {
		prefix = " |   "
	}
	for i, inst := range dep.Instances {
		isLastInst := i == len(dep.Instances)-1
		var prefix2 string
		if isLastInst {
			prefix2 = "└"
		} else {
			prefix2 = "├"
		}
		fmt.Fprintf(w, "%s%s- %s\t%s\t\n", prefix, prefix2, inst.Name, inst.Phase)
	}
}
