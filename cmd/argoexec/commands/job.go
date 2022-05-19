package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

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
			job := tmpl.Job
			steps := job.Steps
			p := wfv1.PendingProgress(len(steps))
			for i, s := range steps {
				p = p.WithStatus(i, wfv1.NodeRunning)
				log := log.WithField("step", i).WithField("p", p)
				ok, err := expr.Eval(s.GetIf(), map[string]interface{}{
					"success": func() bool { return !p.Failure() },
					"failure": func() bool { return p.Failure() },
					"always":  func() bool { return true },
				})
				if err != nil {
					return err
				}
				if ok.(bool) {
					if err := os.WriteFile(os.Getenv(common.EnvVarProgressFile), []byte(p), os.ModePerm); err != nil {
						return err
					}
					log.Info("running step")
					cmd := exec.Command("sh", "-c", s.Run)
					cmd.Stdin = os.Stdin
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					err := cmd.Run()
					log.WithError(err).Info("step complete")
					if err != nil {
						p = p.WithStatus(i, wfv1.NodeFailed)
					} else {
						p = p.WithStatus(i, wfv1.NodeSucceeded)
					}
				} else {
					log.Info("skipped step")
					p = p.WithStatus(i, wfv1.NodeSkipped)
				}
				if err := os.WriteFile(os.Getenv(common.EnvVarProgressFile), []byte(p), os.ModePerm); err != nil {
					return err
				}
			}
			if p.Failure() {
				return fmt.Errorf("failure")
			}
			return nil
		},
	}
}
