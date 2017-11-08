package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"applatix.io/api"
	"applatix.io/axops/service"
	"applatix.io/axops/utils"
	"applatix.io/template"
	humanize "github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

var (
	gitPath string
)

var (
	jobSubmitArgs jobSubmitFlags
	jobListArgs   jobListFlags
	jobShowArgs   jobShowFlags
)

type jobSubmitFlags struct {
	local       bool     // --local
	templateDir string   // --template-dir
	repo        string   // --repo
	branch      string   // --branch
	arguments   []string // --argument
	name        string   // --name
	dryRun      bool     // --dryrun
}

type jobListFlags struct {
	since     string // --since
	submitter string // --submitter
	status    string // --status
	showAll   bool   // --show-all
}

type jobShowFlags struct {
	tree bool // --tree
}

func init() {
	RootCmd.AddCommand(jobCmd)

	jobCmd.AddCommand(jobSubmitCmd)
	jobSubmitCmd.Flags().BoolVar(&jobSubmitArgs.local, "local", false, "Submit the local template")
	jobSubmitCmd.Flags().StringVar(&jobSubmitArgs.templateDir, "template-dir", "", "Directory containing templates")
	jobSubmitCmd.Flags().StringVar(&jobSubmitArgs.repo, "repo", "", "Repository")
	jobSubmitCmd.Flags().StringVar(&jobSubmitArgs.branch, "branch", "", "Branch")
	jobSubmitCmd.Flags().StringSliceVarP(&jobSubmitArgs.arguments, "argument", "a", []string{}, "Arguments")
	jobSubmitCmd.Flags().StringVar(&jobSubmitArgs.name, "name", "", "Name of the job to submit")
	jobSubmitCmd.Flags().BoolVar(&jobSubmitArgs.dryRun, "dryrun", false, "Preview the job create payload before submitting")

	jobCmd.AddCommand(jobListCmd)
	jobListCmd.Flags().StringVar(&jobListArgs.since, "since", "", "List jobs newer than a relative duration (e.g 5m, 1h)")
	jobListCmd.Flags().StringVar(&jobListArgs.submitter, "submitter", "", "Only list jobs submitted by the specified user")
	jobListCmd.Flags().StringVar(&jobListArgs.status, "status", "", "Only list with the specified status")
	jobListCmd.Flags().BoolVar(&jobListArgs.showAll, "show-all", false, "Show all jobs, including completed jobs")

	jobCmd.AddCommand(jobShowCmd)
	jobShowCmd.Flags().BoolVar(&jobShowArgs.tree, "tree", false, "Show job service tree")

	jobCmd.AddCommand(jobKillCmd)

	jobCmd.AddCommand(jobLogsCmd)

	gitPath, _ = exec.LookPath("git")
}

var jobCmd = &cobra.Command{
	Use:   "job",
	Short: "job commands",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}

var jobListCmd = &cobra.Command{
	Use:   "list",
	Short: "List running jobs",
	Run:   jobList,
}

var jobSubmitCmd = &cobra.Command{
	Use:   "submit TEMPLATE_NAME",
	Short: "Submit a job template",
	Run:   jobSubmit,
}

var jobShowCmd = &cobra.Command{
	Use:   "show SERVICE_ID",
	Short: "Show details about a job",
	Run:   jobShow,
}

var jobKillCmd = &cobra.Command{
	Use:   "kill SERVICE_ID",
	Short: "Terminate a job",
	Run:   jobKill,
}

var jobLogsCmd = &cobra.Command{
	Use:   "logs SERVICE_ID",
	Short: "Retrieve logs from a container",
	Run:   jobLogs,
}

var (
	defaultServiceListFields = []string{"id", "name", "username", "launch_time", "status", "status_detail", "failure_path", "policy_id"}
	sinceRegex               = regexp.MustCompile("^(\\d+)([smhd])$")
)

// parseSince parses a since string and returns the time.Time in UTC
func parseSince(s string) time.Time {
	matches := sinceRegex.FindStringSubmatch(jobListArgs.since)
	if len(matches) != 3 {
		log.Fatalf("Invalid since format '%s'. Expected format <duration><unit> (e.g. 3h)\n", jobListArgs.since)
	}
	amount, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		log.Fatalln(err)
	}
	var unit time.Duration
	switch matches[2] {
	case "s":
		unit = time.Second
	case "m":
		unit = time.Minute
	case "h":
		unit = time.Hour
	case "d":
		unit = time.Hour * 24
	}
	ago := unit * time.Duration(amount)
	return time.Now().UTC().Add(-ago)
}

func statusString(svc *service.Service) string {
	statusStr := service.StatusStringMap[svc.Status]
	if svc.Status == utils.ServiceStatusFailed && len(svc.FailurePath) > 0 {
		statusStr = statusStr + " (" + svc.FailurePath[len(svc.FailurePath)-1].(string) + ")"
	}
	return statusStr
}

func submitterString(svc *service.Service) string {
	if svc.PolicyId != "" {
		// The service object currently does not capture the policy name -- just the ID.
		// When we start capturing more policy information, change to display the policy name.
		return "Policy"
	}
	return svc.User
}

func jobList(cmd *cobra.Command, args []string) {
	params := api.ServiceListParams{
		Username: jobListArgs.submitter,
		Fields:   defaultServiceListFields,
	}
	if jobListArgs.since != "" {
		params.MinTime = parseSince(jobListArgs.since)
	}
	if jobListArgs.status != "" {
		params.StatusString = jobListArgs.status
	}
	if !jobListArgs.showAll {
		newTrue := true
		params.IsActive = &newTrue
	}
	initClient()
	jobs, axErr := apiClient.ServiceList(params)
	checkFatal(axErr)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSUBMITTER\tSUBMITTED\tSTATUS")
	for _, svc := range jobs {
		cTime := time.Unix(svc.CreateTime, 0)
		now := time.Now()
		hrTimeDiff := humanize.RelTime(cTime, now, "ago", "later")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", svc.Id, svc.Name, submitterString(svc), hrTimeDiff, statusString(svc))
	}
	w.Flush()
}

func jobSubmit(cmd *cobra.Command, args []string) {
	initClient()
	if len(args) != 1 {
		cmd.HelpFunc()(cmd, args)
		os.Exit(1)
	}
	templateName := args[0]
	repo := getRepo()
	branch := getBranch()
	createParams := api.ServiceCreateParams{
		Name:      jobSubmitArgs.name,
		Arguments: parseArguments(),
		DryRun:    jobSubmitArgs.dryRun,
	}

	if jobSubmitArgs.templateDir != "" {
		// if a template directory is specified during submit, it must mean they want local version
		jobSubmitArgs.local = true
	}

	if jobSubmitArgs.local {
		templateDir := getTemplateDir()
		if templateDir == "" {
			os.Exit(1)
		}
		ctx, _ := buildContextFromDir(templateDir, true)
		result, ok := ctx.Results[templateName]
		if !ok {
			log.Fatalf("Template '%s' not found in %s\n", templateName, templateDir)
		}
		if result.AXErr != nil {
			log.Fatalf("Template '%s' is invalid: %s\n", templateName, result.AXErr)
		}
		switch result.Template.GetType() {
		case template.TemplateTypeContainer, template.TemplateTypeWorkflow, template.TemplateTypeDeployment:
		default:
			log.Fatalf("%s templates are not submittable\n", result.Template.GetType())
		}
		st, axErr := service.EmbedServiceTemplate(result.Template, ctx)
		if axErr != nil {
			log.Fatalf("Failed to convert '%s' to service template: %s\n", templateName, axErr)
		}
		createParams.Template = st
		setSessionInfo(st, createParams.Arguments)
		log.Printf("Submitting local template '%s' from repo: %s, branch: %s", templateName, repo, branch)
	} else {
		tmpl, axErr := apiClient.GetTemplateByName(templateName, repo, branch)
		checkFatal(axErr)
		if tmpl == nil {
			log.Fatalf("Could not find template '%s' in repo: %s, branch: %s", templateName, repo, branch)
		}
		setSessionInfo(tmpl, createParams.Arguments)
		log.Printf("Submitting template %s", tmpl)
		createParams.TemplateID = tmpl.GetID()
	}
	svc, axErr := apiClient.ServiceCreate(createParams)
	checkFatal(axErr)
	// if globalArgs.trace {
	// 	bytes, err := json.MarshalIndent(svc, "", "    ")
	// 	if err != nil {
	// 		log.Fatalf("Failed to marshall '%v': %s\n", svc, err)
	// 	}
	// 	fmt.Printf("Service JSON:\n%s\n", string(bytes))
	// }
	printJob(svc)
}

func humanizeTimestamp(epoch int64) string {
	ts := time.Unix(epoch, 0)
	return fmt.Sprintf("%s (%s)", ts.Format("Mon Jan 02 15:04:05 -0700"), humanize.Time(ts))
}

// humanizeDuration humanizes time.Duration output to a meaningful value,
func humanizeDuration(duration time.Duration) string {
	if duration.Seconds() < 60.0 {
		return fmt.Sprintf("%d seconds", int64(duration.Seconds()))
	}
	if duration.Minutes() < 60.0 {
		remainingSeconds := math.Mod(duration.Seconds(), 60)
		return fmt.Sprintf("%d minutes %d seconds", int64(duration.Minutes()), int64(remainingSeconds))
	}
	if duration.Hours() < 24.0 {
		remainingMinutes := math.Mod(duration.Minutes(), 60)
		remainingSeconds := math.Mod(duration.Seconds(), 60)
		return fmt.Sprintf("%d hours %d minutes %d seconds",
			int64(duration.Hours()), int64(remainingMinutes), int64(remainingSeconds))
	}
	remainingHours := math.Mod(duration.Hours(), 24)
	remainingMinutes := math.Mod(duration.Minutes(), 60)
	remainingSeconds := math.Mod(duration.Seconds(), 60)
	return fmt.Sprintf("%d days %d hours %d minutes %d seconds",
		int64(duration.Hours()/24), int64(remainingHours),
		int64(remainingMinutes), int64(remainingSeconds))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func jobShow(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		cmd.HelpFunc()(cmd, args)
		os.Exit(1)
	}
	initClient()
	svc, axErr := apiClient.ServiceGet(args[0])
	checkFatal(axErr)
	// bytes, err := json.MarshalIndent(svc, "", "    ")
	// if err != nil {
	// 	log.Fatalf("Failed to marshall '%v': %s\n", svc, err)
	// }
	// fmt.Printf("%s\n", string(bytes))
	printJob(svc)
	if jobShowArgs.tree {
		fmt.Println()
		printJobTree(svc)
	}
}

func jobKill(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		cmd.HelpFunc()(cmd, args)
		os.Exit(1)
	}
	initClient()
	axErr := apiClient.ServiceDelete(args[0])
	checkFatal(axErr)
	svc, axErr := apiClient.ServiceGet(args[0])
	checkFatal(axErr)
	printJob(svc)
}

// This struct is the format
type JSONLog struct {
	Log    string `json:"log"`
	Stream string `json:"stream"`
	Time   string `json:"time"`
}

func jobLogs(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		cmd.HelpFunc()(cmd, args)
		os.Exit(1)
	}
	initClient()
	res, axErr := apiClient.ServiceLogs(args[0])
	checkFatal(axErr)
	defer res.Body.Close()
	contentType := res.Header.Get("Content-Type")
	switch contentType {
	case "text/event-stream":
		rd := bufio.NewReader(res.Body)
		for {
			line, err := rd.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return
				}
				log.Fatal("Read Error:", err)
			}
			if !bytes.HasPrefix(line, []byte("data:")) {
				continue
			}
			line = bytes.TrimPrefix(line, []byte("data:"))
			var jl JSONLog
			err = json.Unmarshal(line, &jl)
			if err != nil {
				log.Fatalf("Failed to decode line '%s': %v", string(line), err)
			}
			fmt.Print(jl.Log)
		}
	default:

	}
}

func printJob(svc *service.Service) {
	const svcFmtStr = "%-17s %v\n"
	fmt.Printf(svcFmtStr, "ID:", svc.Id)
	fmt.Printf(svcFmtStr, "URL:", jobURL(svc.Id))
	fmt.Printf(svcFmtStr, "Name:", svc.Name)
	fmt.Printf(svcFmtStr, "Status:", service.StatusStringMap[svc.Status])
	for key, valIf := range svc.StatusDetail {
		if valIf == nil {
			continue
		}
		if val, ok := valIf.(string); ok && val != "" {
			fmt.Printf(svcFmtStr, "  "+key+":", val)
		}
	}
	if svc.Status == utils.ServiceStatusFailed {
		failurePath := make([]string, len(svc.FailurePath))
		for i := range svc.FailurePath {
			failurePath[i] = svc.FailurePath[i].(string)
		}
		if len(failurePath) > 0 {
			fmt.Printf(svcFmtStr, "Failure Path:", strings.Join(failurePath, " -> "))
		}
	}
	fmt.Printf(svcFmtStr, "Submitter:", submitterString(svc))
	fmt.Printf(svcFmtStr, "Submitted:", humanizeTimestamp(svc.CreateTime))
	if svc.LaunchTime > 0 {
		fmt.Printf(svcFmtStr, "Started:", humanizeTimestamp(svc.LaunchTime))
	}
	var duration time.Duration
	if svc.EndTime > 0 {
		fmt.Printf(svcFmtStr, "Completed:", humanizeTimestamp(svc.EndTime))
		duration = time.Second * time.Duration(svc.EndTime-svc.LaunchTime)
	} else if svc.LaunchTime > 0 {
		duration = time.Second * time.Duration(time.Now().Unix()-svc.LaunchTime)
	} else {
		duration = 0
	}
	fmt.Printf(svcFmtStr, "Duration:", humanizeDuration(duration))
	fmt.Printf(svcFmtStr, "Template:", svc.Template.GetName())
	if len(svc.Arguments) > 0 {
		keys := []string{}
		maxLen := 0
		for k := range svc.Arguments {
			keys = append(keys, k)
			maxLen = max(maxLen, len(k))
		}
		fmtStr := "  %-" + strconv.Itoa(maxLen+2) + "s %s\n"
		sort.Strings(keys)
		fmt.Printf(svcFmtStr, "Arguments:", "")
		for _, argName := range keys {
			fmt.Printf(fmtStr, argName+": ", *svc.Arguments[argName])
		}
	}
}

// printJobTree will print out the service tree of a job
func printJobTree(svc *service.Service) {
	statusMap := map[string]int{}
	for _, child := range svc.Children {
		statusMap[child.Id] = child.Status
	}
	statusMap[svc.Id] = svc.Status
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	printJobTreeHelper(w, svc, svc.Name, statusMap, 0, " ", " ")
	w.Flush()
}

var jobStatusIconMap map[int]string

func printJobTreeHelper(w *tabwriter.Writer, svc *service.Service, nodeName string, statusMap map[string]int, depth int, nodePrefix string, childPrefix string) {
	if svc.Template == nil {
		return
	}
	nodeStatus := statusMap[svc.Id]
	nodeName = fmt.Sprintf("%s %s", jobStatusIconMap[nodeStatus], nodeName)

	templateType := svc.Template.GetType()
	var svcID string
	if templateType == template.TemplateTypeContainer {
		svcID = svc.Id
	}

	fmt.Fprintf(w, "%s%s\t%s\n", nodePrefix, nodeName, svcID)
	if len(svc.Children) > 0 {
		fmt.Fprintf(w, " |\n")
	}
	if templateType == template.TemplateTypeWorkflow {
		wt := svc.Template.(*service.EmbeddedWorkflowTemplate)
		for i, parallelSteps := range wt.Steps {
			j := 0
			lastStepGroup := bool(i == len(wt.Steps)-1)
			var part1, subp1 string
			if lastStepGroup {
				part1 = "└-"
				subp1 = "  "
			} else {
				part1 = "├-"
				subp1 = "| "
			}
			// display parallel steps in alphabetical order (for consistency between CLI invocations)
			var keys []string
			for k := range parallelSteps {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, stepName := range keys {
				childSvc := parallelSteps[stepName]
				if j > 0 {
					if lastStepGroup {
						part1 = "  "
					} else {
						part1 = "| "
					}
				}
				firstParallel := bool(j == 0)
				lastParallel := bool(j == len(parallelSteps)-1)
				var part2, subp2 string
				if firstParallel {
					part2 = "·-"
					if !lastParallel {
						subp2 = "| "
					} else {
						subp2 = "  "
					}

				} else if lastParallel {
					part2 = "└-"
					subp2 = "  "
				} else {
					part2 = "├-"
					subp2 = "| "
				}
				childNodePrefix := childPrefix + part1 + part2
				childChldPrefix := childPrefix + subp1 + subp2
				printJobTreeHelper(w, childSvc, stepName, statusMap, depth+1, childNodePrefix, childChldPrefix)
				j = j + 1
			}
		}
	}
}

// jobURL returns the job URL given a job ID
func jobURL(id string) string {
	return fmt.Sprintf("%s/app/timeline/jobs/%s", apiClient.Config.URL, id)
}

// parseArguments parses the -a options supplied in the command line and returns a map from name -> val
func parseArguments() template.Arguments {
	args := make(template.Arguments)
	for _, v := range jobSubmitArgs.arguments {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 1 {
			log.Fatalf("Expected parameter of the form: NAME=VALUE. Recieved: %s", v)
		}
		args[parts[0]] = &parts[1]
	}
	return args
}

// setSessionInfo looks at any input parameters of a service template, and sets %%sesion.repo%% %%sesion.commit%% automatically if it can be determined.
// errors if it cannot be determined. NOTE: this check should be moved to server, which currently silently allows %%session%% variables to execute unresolved
func setSessionInfo(tmpl service.EmbeddedTemplateIf, args template.Arguments) {
	inputs := tmpl.GetInputs()
	if inputs == nil {
		return
	}
	for inParamName, inParam := range inputs.Parameters {
		argName := fmt.Sprintf("parameters.%s", inParamName)
		if _, ok := args[argName]; ok {
			// param was explicitly supplied from command line. nothing to do
			continue
		}
		if inParam == nil || inParam.Default == nil {
			continue
		}
		switch *inParam.Default {
		case "%%session.repo%%":
			if args == nil {
				args = make(template.Arguments)
			}
			repo := getRepo()
			args[fmt.Sprintf("parameters.%s", inParamName)] = &repo
		case "%%session.commit%%":
			if args == nil {
				args = make(template.Arguments)
			}
			branch := getBranch()
			args[fmt.Sprintf("parameters.%s", inParamName)] = &branch
		}
	}
}

// gitRemoteGetURL attempts to determine the repo URL from flags or a `git remote get-url origin` command
func getRepo() string {
	if jobSubmitArgs.repo != "" {
		return jobSubmitArgs.repo
	}
	if gitPath == "" {
		log.Fatal("Failed to automatically determine repo URL: git not found")
	}
	originURL := runCmd(gitPath, "remote", "get-url", "origin")
	originURL = strings.TrimSpace(originURL)
	if strings.HasPrefix(originURL, "git@") {
		// we store repo URLs using https protocol, so convert the URL to an HTTP
		originURL = strings.Replace(originURL, ":", "/", -1)
		originURL = "https://" + strings.TrimPrefix(originURL, "git@")
	}
	return originURL
}

// gitGetBranch attempts to determine the branch based on a `git rev-parse --abbrev-ref HEAD` command
func getBranch() string {
	if jobSubmitArgs.branch != "" {
		return jobSubmitArgs.branch
	}
	if gitPath == "" {
		log.Fatal("Failed to automatically determine branch: git not found")
	}
	branch := runCmd(gitPath, "rev-parse", "--abbrev-ref", "HEAD")
	return strings.TrimSpace(branch)
}

// getRemoteAndBranch attempts to get the remote ref and branch for the current HEAD in the local repo
func getRemoteAndBranch() (string, string) {
	if gitPath == "" {
		log.Fatal("Failed to automatically determine remote and branch: git not found")
	}
	remoteAndBranch := runCmd(gitPath, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	tmpArr := strings.Split(strings.TrimSpace(remoteAndBranch), "/")
	remoteURL := runCmd(gitPath, "config", "--get", fmt.Sprintf("remote.%s.url", tmpArr[0]))
	if strings.HasPrefix(remoteURL, "git@") {
		// we store repo URLs using https protocol, so convert the URL to an HTTP
		remoteURL = strings.Replace(remoteURL, ":", "/", -1)
		remoteURL = "https://" + strings.TrimPrefix(remoteURL, "git@")
	}
	return strings.TrimSpace(remoteURL), tmpArr[1]
}

// getTemplateDir attempts to find the .argo directory based on current path
func getTemplateDir() string {
	if jobSubmitArgs.templateDir != "" {
		return jobSubmitArgs.templateDir
	}
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		tmplDir := path.Join(dir, ".argo")
		fileInfo, err := os.Stat(tmplDir)
		if err == nil && fileInfo.IsDir() {
			return tmplDir
		}
		if dir == "/" {
			return ""
		}
		dir = path.Clean(dir + "/..")
	}
}
