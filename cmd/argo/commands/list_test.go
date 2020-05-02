package commands

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func getListArgs() listFlags {
	return listFlags{
		allNamespaces: false,
		status:        []string{},
		completed:     false,
		running:       false,
		prefix:        "",
		output:        "wide",
		since:         "",
		chunkSize:     500,
		noHeaders:     false,
		continueToken: "",
		limit:         500,
	}
}

func getWorkflowList() wfv1.WorkflowList {
	wfList := wfv1.WorkflowList{
		Items: []wfv1.Workflow{},
	}
	now := time.Now()
	for i := 0; i < 3; i++ {
		wfList.Items = append(wfList.Items, wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("my-wf-%d", i), Namespace: "my-ns", CreationTimestamp: metav1.Time{Time: now}},
			Spec: wfv1.WorkflowSpec{
				Arguments: wfv1.Arguments{Parameters: []wfv1.Parameter{
					{Name: "my-param", Value: pointer.StringPtr("my-value")},
				}},
				Priority: pointer.Int32Ptr(2),
				Templates: []wfv1.Template{
					{Name: "t0", Container: &corev1.Container{}},
				},
			},
			Status: wfv1.WorkflowStatus{
				Phase:      wfv1.NodeRunning,
				StartedAt:  metav1.Time{Time: now},
				FinishedAt: metav1.Time{Time: now.Add(3 * time.Second)},
				Nodes: wfv1.Nodes{
					"n0": {Phase: wfv1.NodePending, Type: wfv1.NodeTypePod, TemplateName: "t0"},
					"n1": {Phase: wfv1.NodeRunning, Type: wfv1.NodeTypePod, TemplateName: "t0"},
					"n2": {Phase: wfv1.NodeRunning, Type: wfv1.NodeTypePod, TemplateName: "t0"},
					"n3": {Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod, TemplateName: "t0"},
					"n4": {Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod, TemplateName: "t0"},
					"n5": {Phase: wfv1.NodeError, Type: wfv1.NodeTypePod, TemplateName: "t0"},
				},
			},
		},
		)
	}
	return wfList
}

// TestGetListOpts
func TestGetListOpts(t *testing.T) {
	listArgs := getListArgs()
	opts := getListOpts(&listArgs)
	assert.Equal(t, "", opts.LabelSelector)

	listArgs.status = append(listArgs.status, "RUNNING")
	listArgs.completed = true
	opts = getListOpts(&listArgs)
	assert.Equal(t, "workflows.argoproj.io/completed=true,workflows.argoproj.io/phase in (RUNNING)", opts.LabelSelector)

	listArgs.completed = false
	listArgs.running = true
	opts = getListOpts(&listArgs)
	assert.Equal(t, "workflows.argoproj.io/completed!=true,workflows.argoproj.io/phase in (RUNNING)", opts.LabelSelector)
}

// TestGetKubeCursor
func TestGetKubeCursor(t *testing.T) {
	listArgs := getListArgs()
	cursor, wfName, err := getKubeCursor(&listArgs)
	if assert.Nil(t, err) {
		assert.Equal(t, "", cursor)
		assert.Equal(t, "", wfName)
	}

	listArgs.continueToken = "BLAH"
	cursor, wfName, err = getKubeCursor(&listArgs)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "malformed value")
		assert.Equal(t, "", cursor)
		assert.Equal(t, "", wfName)
	}

	// decoded: Hello World
	listArgs.continueToken = "SGVsbG8gd29ybGQ"
	cursor, wfName, err = getKubeCursor(&listArgs)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "malformed value")
		assert.Equal(t, "", cursor)
		assert.Equal(t, "", wfName)
	}

	// decoded: "{\"kube_cursor\":\"foo\"}"
	listArgs.continueToken = "eyJrdWJlX2N1cnNvciI6ImZvbyJ9"
	cursor, wfName, err = getKubeCursor(&listArgs)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "malformed value")
		assert.Equal(t, "", cursor)
		assert.Equal(t, "", wfName)
	}

	// decoded: "{\"last_workflow_name\":\"foo\",\"prefix\":\"bar\"}"
	listArgs.continueToken = "eyJsYXN0X3dvcmtmbG93X25hbWUiOiJmb28iLCJwcmVmaXgiOiJiYXIifQ"
	cursor, wfName, err = getKubeCursor(&listArgs)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "using the identical values for `prefix` and `since`")
		assert.Equal(t, "", cursor)
		assert.Equal(t, "", wfName)
	}

	// decoded: "{\"last_workflow_name\":\"foo\",\"since\":\"bar\"}"
	listArgs.continueToken = "eyJsYXN0X3dvcmtmbG93X25hbWUiOiJmb28iLCJzaW5jZSI6ImJhciJ9"
	cursor, wfName, err = getKubeCursor(&listArgs)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "using the identical values for `prefix` and `since`")
		assert.Equal(t, "", cursor)
		assert.Equal(t, "", wfName)
	}

	// decoded: "{\"last_workflow_name\":\"foo\",\"kube_cursor\":\"bar\"}"
	listArgs.continueToken = "eyJsYXN0X3dvcmtmbG93X25hbWUiOiJmb28iLCJrdWJlX2N1cnNvciI6ImJhciJ9"
	cursor, wfName, err = getKubeCursor(&listArgs)
	if assert.Nil(t, err) {
		assert.Equal(t, "bar", cursor)
		assert.Equal(t, "foo", wfName)
	}
}

// TestPrintCursor
func TestPrintCursor(t *testing.T) {
	listArgs := getListArgs()
	var buf bytes.Buffer

	printCursor("", "foo", &listArgs, &buf)
	assert.Contains(t, buf.String(), "There are additional suppressed results")

	buf.Reset()
	printCursor("", "", &listArgs, &buf)
	assert.Equal(t, "", buf.String())
}

// TestTruncateWorkflowList
func TestTruncateWorkflowList(t *testing.T) {
	wfList := getWorkflowList()
	listArgs := getListArgs()
	listArgs.limit = 3
	var workflows wfv1.Workflows
	lastWfName := truncateWorkflowList(&wfList, &workflows, &listArgs)
	assert.Equal(t, "my-wf-2", lastWfName)
}

// TestFilterByPrefix
func TestFilterByPrefix(t *testing.T) {
	wfList := getWorkflowList()
	wf := wfList.Items[0]
	assert.True(t, filterByPrefix(&wf, ""))
	assert.True(t, filterByPrefix(&wf, "my-wf"))
	assert.False(t, filterByPrefix(&wf, "foo"))
}

// TestFilterBySince
func TestFilterBySince(t *testing.T) {
	wfList := getWorkflowList()
	wf := wfList.Items[0]

	assert.True(t, filterBySince(&wf, nil))

	ts := wf.ObjectMeta.CreationTimestamp.Add(-1)
	assert.True(t, filterBySince(&wf, &ts))

	ts = time.Now()
	wf.Status.FinishedAt.Reset()
	assert.True(t, filterBySince(&wf, &ts))
}
