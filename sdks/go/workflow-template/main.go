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
	"k8s.io/client-go/tools/clientcmd"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	v1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
)

func main() {
	var (
		kubeconfig = flag.String("kubeconfig", defaultKubeconfig(), "path to kubeconfig file")
		namespace  = flag.String("namespace", "argo", "namespace for resources")
	)
	flag.Parse()

	ctx := context.Background()

	// Load kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading kubeconfig: %v\n", err)
		os.Exit(1)
	}

	clientset := wfclientset.NewForConfigOrDie(config)

	// Get clients
	wftClient := clientset.ArgoprojV1alpha1().WorkflowTemplates(*namespace)
	wfClient := clientset.ArgoprojV1alpha1().Workflows(*namespace)

	fmt.Println("=== Workflow Template Example ===")

	// Step 1: Create a WorkflowTemplate
	fmt.Printf("Step 1: Creating WorkflowTemplate...\n")
	// <snip id="create-workflow-template">
	template := &wfv1.WorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hello-world",
			Labels: map[string]string{
				"example": "workflow-template",
			},
		},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "greet",
			Arguments: wfv1.Arguments{
				Parameters: []wfv1.Parameter{
					{
						Name:  "message",
						Value: wfv1.AnyStringPtr("Hello World"),
					},
				},
			},
			Templates: []wfv1.Template{
				{
					Name: "greet",
					Inputs: wfv1.Inputs{
						Parameters: []wfv1.Parameter{
							{Name: "message"},
						},
					},
					Container: &corev1.Container{
						Image:   "busybox:latest",
						Command: []string{"echo"},
						Args:    []string{"{{inputs.parameters.message}}"},
					},
				},
			},
		},
	}

	var createdTemplate *wfv1.WorkflowTemplate
	existingTemplate, err := wftClient.Get(ctx, template.Name, metav1.GetOptions{})
	if err == nil {
		template.ResourceVersion = existingTemplate.ResourceVersion
		createdTemplate, err = wftClient.Update(ctx, template, metav1.UpdateOptions{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating template: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ WorkflowTemplate '%s' updated (already existed)\n\n", createdTemplate.Name)
	} else {
		createdTemplate, err = wftClient.Create(ctx, template, metav1.CreateOptions{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating template: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ WorkflowTemplate '%s' created\n\n", createdTemplate.Name)
	}
	// </snip>

	// Step 2: Submit workflow from template with default parameters
	fmt.Printf("Step 2: Submitting workflow from template (default params)...\n")
	// <snip id="submit-from-template">
	workflow1 := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "from-template-default-",
		},
		Spec: wfv1.WorkflowSpec{
			WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{
				Name: createdTemplate.Name,
			},
		},
	}

	submitted1, err := wfClient.Create(ctx, workflow1, metav1.CreateOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error submitting workflow: %v\n", err)
		cleanup(ctx, wftClient, createdTemplate.Name)
		os.Exit(1)
	}
	fmt.Printf("✓ Workflow '%s' submitted with default parameters\n\n", submitted1.Name)
	// </snip>

	// Step 3: Submit workflow with custom parameters
	fmt.Printf("Step 3: Submitting workflow with custom parameters...\n")
	workflow2 := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "from-template-custom-",
		},
		Spec: wfv1.WorkflowSpec{
			WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{
				Name: createdTemplate.Name,
			},
			Arguments: wfv1.Arguments{
				Parameters: []wfv1.Parameter{
					{
						Name:  "message",
						Value: wfv1.AnyStringPtr("Custom greeting from Go SDK!"),
					},
				},
			},
		},
	}

	submitted2, err := wfClient.Create(ctx, workflow2, metav1.CreateOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error submitting workflow: %v\n", err)
		cleanup(ctx, wftClient, createdTemplate.Name)
		os.Exit(1)
	}
	fmt.Printf("✓ Workflow '%s' submitted with custom message\n\n", submitted2.Name)

	// Step 4: List workflows created from this template
	time.Sleep(time.Second) // Hopefully enough time for the list to return the newly created objects
	fmt.Printf("Step 4: Listing workflows created from template...\n")
	labelSelector := fmt.Sprintf("workflows.argoproj.io/workflow-template=%s", createdTemplate.Name)
	list, err := wfClient.List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing workflows: %v\n", err)
	} else {
		fmt.Printf("Found %d workflow(s) from template '%s':\n", len(list.Items), createdTemplate.Name)
		for i, wf := range list.Items {
			phase := wf.Status.Phase
			if phase == "" {
				phase = "Pending"
			}
			fmt.Printf("  %d. %s (%s)\n", i+1, wf.Name, phase)
		}
	}

	// Step 5: Get template details
	fmt.Printf("\nStep 5: Template details:\n")
	fetchedTemplate, err := wftClient.Get(ctx, createdTemplate.Name, metav1.GetOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting template: %v\n", err)
	} else {
		fmt.Printf("  Name: %s\n", fetchedTemplate.Name)
		fmt.Printf("  Entrypoint: %s\n", fetchedTemplate.Spec.Entrypoint)
		fmt.Printf("  Templates: %d\n", len(fetchedTemplate.Spec.Templates))
		fmt.Printf("  Parameters:\n")
		for _, param := range fetchedTemplate.Spec.Arguments.Parameters {
			value := ""
			if param.Value != nil {
				value = param.Value.String()
			}
			fmt.Printf("    - %s = %s\n", param.Name, value)
		}
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("✓ Created WorkflowTemplate: %s\n", createdTemplate.Name)
	fmt.Printf("✓ Submitted 2 workflows from template\n")
	fmt.Printf("\nView workflows with:\n")
	fmt.Printf("  argo get %s -n %s\n", submitted1.Name, *namespace)
	fmt.Printf("  argo get %s -n %s\n", submitted2.Name, *namespace)
	fmt.Printf("\nCleanup with:\n")
	fmt.Printf("  kubectl delete workflowtemplate %s -n %s\n", createdTemplate.Name, *namespace)
	fmt.Printf("  kubectl delete workflow %s %s -n %s\n", submitted1.Name, submitted2.Name, *namespace)
}

func cleanup(ctx context.Context, wftClient v1alpha1.WorkflowTemplateInterface, templateName string) {
	fmt.Printf("\nCleaning up template '%s'...\n", templateName)
	if err := wftClient.Delete(ctx, templateName, metav1.DeleteOptions{}); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting template: %v\n", err)
	}
}

func defaultKubeconfig() string {
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".kube", "config")
	}
	return ""
}
