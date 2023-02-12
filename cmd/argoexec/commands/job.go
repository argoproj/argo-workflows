package commands

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/antonmedv/expr"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func NewJobCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "job",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			tmpl := &wfv1.Template{}
			if err := json.Unmarshal([]byte(os.Getenv(common.EnvVarTemplate)), tmpl); err != nil {
				return err
			}
			var finalErr error
			for _, step := range tmpl.Job.Steps {
				stepPhase := wfv1.NodeRunning
				failed := finalErr != nil
				log := log.WithField("step", step.Name).WithField("failed", failed)
				ok, err := expr.Eval(step.GetIf(), map[string]interface{}{
					"success": func() bool { return !failed },
					"failure": func() bool { return failed },
					"always":  func() bool { return true },
				})
				if err != nil {
					return err
				}
				if ok.(bool) {
					log.Info("running step")
					cmd := exec.Command("sh", "-c", step.Run)
					cmd.Env = os.Environ()
					cmd.Stdin = os.Stdin
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					err := cmd.Run()
					log.WithError(err).Info("step complete")
					if err != nil {
						if finalErr == nil {
							finalErr = err
						}
						stepPhase = wfv1.NodeFailed
					} else {
						stepPhase = wfv1.NodeSucceeded
					}
				} else {
					log.Info("skipped step")
					stepPhase = wfv1.NodeSkipped
				}
				filename := filepath.Join(common.VarRunArgoPath, step.Name, "phase")
				_ = os.Mkdir(filepath.Dir(filename), os.ModePerm)
				if err := os.WriteFile(filename, []byte(stepPhase), os.ModePerm); err != nil {
					return err
				}
				if stepPhase.FailedOrError() {
					failed = true
				}
			}
			return finalErr
		},
	}
}
