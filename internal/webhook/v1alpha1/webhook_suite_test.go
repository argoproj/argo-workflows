package v1alpha1

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	argoprojiov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// +kubebuilder:scaffold:imports
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cancel            context.CancelFunc
	cfg               *rest.Config
	ctx               context.Context
	k8sClient         client.Client
	testEnv           *envtest.Environment
	testNamespaceName = "test-ns"
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Webhook Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "manifests", "base", "crds", "minimal")},
		ErrorIfCRDPathMissing: true,

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "..", "..", "bin", "k8s",
			fmt.Sprintf("1.31.0-%s-%s", runtime.GOOS, runtime.GOARCH)),

		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "..", "manifests", "base", "webhook")},
		},
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	scheme := apimachineryruntime.NewScheme()
	err = corev1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = argoprojiov1alpha1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = admissionv1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
	err = k8sClient.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespaceName,
		},
	})
	Expect(err).ToNot(HaveOccurred())

	// start webhook server using Manager.
	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    webhookInstallOptions.LocalServingHost,
			Port:    webhookInstallOptions.LocalServingPort,
			CertDir: webhookInstallOptions.LocalServingCertDir,
		}),
		LeaderElection: false,
		Metrics:        metricsserver.Options{BindAddress: "0"},
	})
	Expect(err).NotTo(HaveOccurred())

	err = SetupWorkflowWebhookWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:webhook

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()

	// wait for the webhook server to get ready.
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)
	Eventually(func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return err
		}

		return conn.Close()
	}).Should(Succeed())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// test helper
func validateOperation(shouldErr bool, errRegexp, operation string, obj client.Object) {
	var err error
	switch operation {
	case "CREATE":
		err = k8sClient.Create(ctx, obj)
	case "UPDATE":
		err = k8sClient.Update(ctx, obj)
	}
	if shouldErr {
		Expect(err).To(HaveOccurred())
		if errRegexp != "" {
			Expect(err.Error()).To(MatchRegexp(errRegexp))
		}
	} else {
		Expect(err).ToNot(HaveOccurred())
	}
}

// test data
const (
	testWorkflowTemplateName        = "test-wft-v1.0.0"
	testClusterWorkflowTemplateName = "test-cwft-v1.0.0"
)

// setup custom resources
var _ = BeforeEach(func() {
	wfSpec := argoprojiov1alpha1.WorkflowSpec{
		Entrypoint: "entrypoint",
		Templates: []argoprojiov1alpha1.Template{
			{
				Name: "entrypoint",
				Steps: []argoprojiov1alpha1.ParallelSteps{
					{
						Steps: []argoprojiov1alpha1.WorkflowStep{
							{
								Name: "main-step",
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
	}
	err := k8sClient.Create(ctx, &argoprojiov1alpha1.WorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testWorkflowTemplateName,
			Namespace: testNamespaceName,
		},
		Spec: wfSpec,
	})
	Expect(err).ToNot(HaveOccurred())
	err = k8sClient.Create(ctx, &argoprojiov1alpha1.ClusterWorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: testClusterWorkflowTemplateName},
		Spec:       wfSpec,
	})
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterEach(func() {
	err := k8sClient.Delete(ctx, &argoprojiov1alpha1.WorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testWorkflowTemplateName,
			Namespace: testNamespaceName,
		},
	})
	Expect(err).ToNot(HaveOccurred())
	err = k8sClient.Delete(ctx, &argoprojiov1alpha1.ClusterWorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: testClusterWorkflowTemplateName},
	})
	Expect(err).ToNot(HaveOccurred())
})
