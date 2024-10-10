package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/upper/db/v4"
	mysqladp "github.com/upper/db/v4/adapter/mysql"
	postgresqladp "github.com/upper/db/v4/adapter/postgresql"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/uuid"

	"github.com/argoproj/argo-workflows/v3/persist/sqldb"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var session db.Session

func main() {
	var dsn string
	rootCmd := &cobra.Command{
		Use:   "db",
		Short: "CLI for developers to use when working on the DB locally",
	}
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) (err error) {
		session, err = createDBSession(dsn)
		return
	}
	rootCmd.PersistentFlags().StringVarP(&dsn, "dsn", "c", "postgres://postgres@localhost:5432/postgres", "DSN connection string. For MySQL, use 'mysql:password@tcp/argo'.")
	rootCmd.AddCommand(NewMigrateCommand())
	rootCmd.AddCommand(NewFakeDataCommand())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func NewMigrateCommand() *cobra.Command {
	var cluster, table string
	migrationCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Force DB migration for given cluster/table",
		RunE: func(cmd *cobra.Command, args []string) error {
			return sqldb.NewMigrate(session, cluster, table).Exec(context.Background())
		},
	}
	migrationCmd.Flags().StringVar(&cluster, "cluster", "default", "Cluster name")
	migrationCmd.Flags().StringVar(&table, "table", "argo_workflows", "Table name")
	return migrationCmd
}

func NewFakeDataCommand() *cobra.Command {
	var template string
	var seed, rows, numClusters, numNamespaces int
	fakeDataCmd := &cobra.Command{
		Use:   "fake-archived-workflows",
		Short: "Insert randomly-generated workflows into argo_archived_workflows, for testing purposes",
		RunE: func(cmd *cobra.Command, args []string) error {
			rand.Seed(int64(seed))
			clusters := randomStringArray(numClusters)
			namespaces := randomStringArray(numNamespaces)
			fmt.Printf("Using seed %d\nClusters: %v\nNamespaces: %v\n", seed, clusters, namespaces)

			instanceIDService := instanceid.NewService("")
			wfTmpl := wfv1.MustUnmarshalWorkflow(template)

			for i := 0; i < rows; i++ {
				wf := randomizeWorkflow(wfTmpl, namespaces)
				cluster := clusters[rand.Intn(len(clusters))]
				wfArchive := sqldb.NewWorkflowArchive(session, cluster, "", instanceIDService)
				if err := wfArchive.ArchiveWorkflow(wf); err != nil {
					return err
				}
			}
			fmt.Printf("Inserted %d rows\n", rows)
			return nil
		},
	}
	fakeDataCmd.Flags().StringVar(&template, "template", "@workflow/controller/testdata/dag-exhausted-retries-xfail.yaml", "Workflow definition to use as a template")
	fakeDataCmd.Flags().IntVar(&seed, "seed", rand.Int(), "Random number seed")
	fakeDataCmd.Flags().IntVar(&rows, "rows", 10, "Number of rows to insert")
	fakeDataCmd.Flags().IntVar(&numClusters, "clusters", 1, "Number of cluster names to autogenerate")
	fakeDataCmd.Flags().IntVar(&numNamespaces, "namespaces", 5, "Number of namespaces to autogenerate")
	return fakeDataCmd
}

func createDBSession(dsn string) (db.Session, error) {
	if strings.HasPrefix(dsn, "postgres") {
		url, err := postgresqladp.ParseURL(dsn)
		if err != nil {
			return nil, err
		}
		return postgresqladp.Open(url)
	} else {
		url, err := mysqladp.ParseURL(dsn)
		if err != nil {
			return nil, err
		}
		return mysqladp.Open(url)
	}
}

func randomStringArray(length int) []string {
	var result []string
	for i := 0; i < length; i++ {
		result = append(result, rand.String(rand.IntnRange(5, 20)))
	}
	return result
}

func randomPhase() wfv1.WorkflowPhase {
	phases := []wfv1.WorkflowPhase{
		wfv1.WorkflowSucceeded,
		wfv1.WorkflowFailed,
		wfv1.WorkflowError,
	}
	return phases[rand.Intn(len(phases))]
}

func randomTime() time.Time {
	min := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	return time.Unix(rand.Int63nRange(min, max), 0)
}

func randomizeWorkflow(wfTmpl *wfv1.Workflow, namespaces []string) *wfv1.Workflow {
	wf := wfTmpl.DeepCopy()
	wf.Name = rand.String(rand.IntnRange(10, 30))
	wf.Namespace = namespaces[rand.Intn(len(namespaces))]
	wf.UID = uuid.NewUUID()
	wf.Status.Phase = randomPhase()
	startTime := randomTime()
	wf.Status.StartedAt = metav1.NewTime(startTime)
	wf.Status.FinishedAt = metav1.NewTime(startTime.Add(time.Second * time.Duration(rand.IntnRange(5, 300))))
	if wf.Labels == nil {
		wf.Labels = map[string]string{}
	}
	wf.Labels["workflows.argoproj.io/phase"] = string(wf.Status.Phase)
	return wf
}
