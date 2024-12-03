package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	argoprojiov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Workflow Webhook", func() {
	var (
		obj       *argoprojiov1alpha1.Workflow
		oldObj    *argoprojiov1alpha1.Workflow
		validator WorkflowCustomValidator
	)

	BeforeEach(func() {
		obj = &argoprojiov1alpha1.Workflow{}
		oldObj = &argoprojiov1alpha1.Workflow{}
		validator = WorkflowCustomValidator{}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
		// TODO (user): Add any setup logic common to all tests
	})

	Context("When creating or updating Workflow under Validating Webhook", func() {
		DescribeTable("validate workflow resources",
			func(shouldErr bool, errRegexp, operation string, obj client.Object) {
				validateOperation(shouldErr, errRegexp, operation, obj)
			},
			Entry(
				"deny invalid workflow with invalid step name",
				true,
				`.+workflow failed validation: templates\.entrypoint.steps\[0\]\.name \'bad\.name\' is invalid.+`,
				"CREATE",
				&argoprojiov1alpha1.Workflow{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "test-foo",
						Namespace:    testNamespaceName,
					},
					Spec: argoprojiov1alpha1.WorkflowSpec{
						Entrypoint: "entrypoint",
						Templates: []argoprojiov1alpha1.Template{
							{
								Name: "entrypoint",
								Steps: []argoprojiov1alpha1.ParallelSteps{
									{
										Steps: []argoprojiov1alpha1.WorkflowStep{
											{
												Name: "bad.name",
												Inline: &argoprojiov1alpha1.Template{
													Name: "inline",
													Container: &corev1.Container{
														Name:    "main-container",
														Image:   "whalesay",
														Command: []string{"launcher"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			),
			Entry(
				"deny invalid workflow with invalid templateRef",
				true,
				`.+workflow failed validation: templates\.entrypoint\.tasks\.invalid-step template reference entrypoint\.some-template not found`,
				"CREATE",
				&argoprojiov1alpha1.Workflow{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "test-foo",
						Namespace:    testNamespaceName,
					},
					Spec: argoprojiov1alpha1.WorkflowSpec{
						Entrypoint: "entrypoint",
						Templates: []argoprojiov1alpha1.Template{
							{
								Name: "entrypoint",
								DAG: &argoprojiov1alpha1.DAGTemplate{
									Tasks: []argoprojiov1alpha1.DAGTask{
										{
											Name: "invalid-step",
											TemplateRef: &argoprojiov1alpha1.TemplateRef{
												Template:     "some-template",
												Name:         "entrypoint",
												ClusterScope: false,
											},
										},
									},
								},
							},
						},
					},
				},
			),
			Entry(
				"deny invalid workflow with invalid cluster scope templateRef",
				true,
				`.+workflow failed validation: templates\.entrypoint\.tasks\.invalid-step template reference entrypoint\.some-cluster-template not found`,
				"CREATE",
				&argoprojiov1alpha1.Workflow{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "test-foo",
						Namespace:    testNamespaceName,
					},
					Spec: argoprojiov1alpha1.WorkflowSpec{
						Entrypoint: "entrypoint",
						Templates: []argoprojiov1alpha1.Template{
							{
								Name: "entrypoint",
								DAG: &argoprojiov1alpha1.DAGTemplate{
									Tasks: []argoprojiov1alpha1.DAGTask{
										{
											Name: "invalid-step",
											TemplateRef: &argoprojiov1alpha1.TemplateRef{
												Template:     "some-cluster-template",
												Name:         "entrypoint",
												ClusterScope: true,
											},
										},
									},
								},
							},
						},
					},
				},
			),
			Entry(
				"allow valid workflow", false, "", "CREATE", &argoprojiov1alpha1.Workflow{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "test-foo",
						Namespace:    testNamespaceName,
					},
					Spec: argoprojiov1alpha1.WorkflowSpec{
						Entrypoint: "entrypoint",
						Templates: []argoprojiov1alpha1.Template{
							{
								Name: "entrypoint",
								Steps: []argoprojiov1alpha1.ParallelSteps{
									{
										Steps: []argoprojiov1alpha1.WorkflowStep{
											{
												Name: "good-name",
												Inline: &argoprojiov1alpha1.Template{
													Name: "inline",
													Container: &corev1.Container{
														Name:    "main-container",
														Image:   "whalesay",
														Command: []string{"launcher"},
													},
												},
											},
											{
												Name: "good-template-ref",
												TemplateRef: &argoprojiov1alpha1.TemplateRef{
													Template:     "entrypoint",
													Name:         testWorkflowTemplateName,
													ClusterScope: false,
												},
											},
											{
												Name: "good-cluster-template-ref",
												TemplateRef: &argoprojiov1alpha1.TemplateRef{
													Template:     "entrypoint",
													Name:         testClusterWorkflowTemplateName,
													ClusterScope: true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			),
		)
	})
})
