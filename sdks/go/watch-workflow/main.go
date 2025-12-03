package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	v1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
)

func main() {
	var (
		kubeconfig = flag.String("kubeconfig", defaultKubeconfig(), "path to kubeconfig file")
		namespace  = flag.String("namespace", "argo", "namespace for workflow")
	)
	flag.Parse()

	ctx := context.Background()

	// Load kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading kubeconfig: %v\n", err)
		os.Exit(1)
	}

	// Create workflow client
	wfClient := wfclientset.NewForConfigOrDie(config).
		ArgoprojV1alpha1().
		Workflows(*namespace)

	// Define a workflow that takes a few seconds to complete
	workflow := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "watch-example-",
		},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "sleep-and-echo",
			Templates: []wfv1.Template{
				{
					Name: "sleep-and-echo",
					Container: &corev1.Container{
						Image:   "busybox:latest",
						Command: []string{"sh", "-c"},
						Args:    []string{"echo 'Starting...'; sleep 5; echo 'Done!'"},
					},
				},
			},
		},
	}

	// Submit the workflow
	fmt.Printf("Submitting workflow...\n")
	created, err := wfClient.Create(ctx, workflow, metav1.CreateOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating workflow: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Workflow %s submitted\n", created.Name)
	fmt.Printf("\nWatching workflow progress...\n")
	fmt.Println("─────────────────────────────────────────────")

	// Watch the workflow until completion
	if err := watchWorkflow(ctx, wfClient, created.Name); err != nil {
		fmt.Fprintf(os.Stderr, "Error watching workflow: %v\n", err)
		os.Exit(1)
	}
}

// <snip id="watch-workflow">
func watchWorkflow(ctx context.Context, wfClient v1alpha1.WorkflowInterface, name string) error {
	// Create field selector to watch only this workflow
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	// Start watching with a timeout
	watchIf, err := wfClient.Watch(ctx, metav1.ListOptions{
		FieldSelector:   fieldSelector.String(),
		TimeoutSeconds:  ptr.To(int64(300)), // 5 minute timeout
		ResourceVersion: "0",
	})
	if err != nil {
		return fmt.Errorf("failed to watch workflow: %w", err)
	}
	defer watchIf.Stop()

	// Track last seen phase to avoid duplicate messages
	lastPhase := wfv1.WorkflowUnknown
	startTime := time.Now()

	// Process watch events
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case event, ok := <-watchIf.ResultChan():
			if !ok {
				return fmt.Errorf("watch channel closed unexpectedly")
			}

			wf, ok := event.Object.(*wfv1.Workflow)
			if !ok {
				continue
			}

			switch event.Type {
			case watch.Added:
				fmt.Printf("[%s] Workflow created\n", formatDuration(time.Since(startTime)))

			case watch.Modified:
				// Only print if phase changed
				if wf.Status.Phase != lastPhase {
					lastPhase = wf.Status.Phase
					fmt.Printf("[%s] Phase: %s\n", formatDuration(time.Since(startTime)), wf.Status.Phase)

					// Print additional details based on phase
					if wf.Status.Phase == wfv1.WorkflowRunning && !wf.Status.StartedAt.IsZero() {
						fmt.Printf("         Started at: %s\n", wf.Status.StartedAt.Format(time.RFC3339))
					}
				}

				// Check if workflow is complete
				if !wf.Status.FinishedAt.IsZero() {
					fmt.Println("─────────────────────────────────────────────")
					fmt.Printf("✓ Workflow completed!\n")
					fmt.Printf("  Final Phase: %s\n", wf.Status.Phase)
					fmt.Printf("  Started: %s\n", wf.Status.StartedAt.Format(time.RFC3339))
					fmt.Printf("  Finished: %s\n", wf.Status.FinishedAt.Format(time.RFC3339))
					fmt.Printf("  Duration: %s\n", wf.Status.FinishedAt.Sub(wf.Status.StartedAt.Time))

					if wf.Status.Message != "" {
						fmt.Printf("  Message: %s\n", wf.Status.Message)
					}

					// Print node statuses
					if len(wf.Status.Nodes) > 0 {
						fmt.Printf("\nNode Details:\n")
						for nodeName, node := range wf.Status.Nodes {
							fmt.Printf("  - %s: %s\n", nodeName, node.Phase)
						}
					}

					return nil
				}

			case watch.Deleted:
				fmt.Printf("[%s] Workflow deleted\n", formatDuration(time.Since(startTime)))
				return nil
			}
		}
	}
}
// </snip>

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	return fmt.Sprintf("%02d:%02d", int(d.Minutes()), int(d.Seconds())%60)
}

func defaultKubeconfig() string {
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".kube", "config")
	}
	return ""
}
