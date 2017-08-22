package utils

import (
	"strconv"
	"strings"
)

// Environment key words
const (
	// namespaces
	EnvNSSession         = "session"
	EnvNSService         = "service"
	EnvNSStep            = "steps" // alias for EnvNSService
	EnvNSFixture         = "fixtures"
	EnvNSArtifact        = "artifacts"
	EnvNSTag             = "tag"
	EnvNSWorkflow        = "workflow"
	EnvNSSecret          = "secrets"
	EnvNSVolume          = "volumes"
	EnvNSTemplateFixture = "fixture"

	// Environment keys
	EnvKeyUser         = "user"
	EnvKeyRepo         = "repo"
	EnvKeyBranch       = "branch"
	EnvKeyCommit       = "commit"
	EnvKeyTargetBranch = "target_branch"
	EnvKeyId           = "id"
)

// ServiceStatus status
const (
	ServiceStatusSuccess    = 0
	ServiceStatusWaiting    = 1
	ServiceStatusRunning    = 2
	ServiceStatusCanceling  = 3
	ServiceStatusFailed     = -1
	ServiceStatusCancelled  = -2
	ServiceStatusSkipped    = -3
	ServiceStatusInitiating = 255
)

const (
	//Notify when task is started
	ServiceEventOnStart = "on_start"
	//Notify when task is successful
	ServiceEventOnSuccess = "on_success"
	//Notify when task is failed
	ServiceEventOnFailure = "on_failure"
	//Notify when task status is changed, eg. task is failed for the first time, task is not failing any more
	ServiceEventOnChange = "on_change"
	//Notify when task completes (success, failure, cancelled, skipped)
	ServiceEventOnCompletion = "on_completion"
)

var ServiceEventMap = map[string]bool{
	ServiceEventOnStart:      true,
	ServiceEventOnSuccess:    true,
	ServiceEventOnFailure:    true,
	ServiceEventOnChange:     true,
	ServiceEventOnCompletion: true,
}

type EC2Price struct {
	Cpu      int
	Mem      float64
	Cost     float64 // cents per second for the instance
	CoreCost float64 // per core cents per second
	MemCost  float64 // per MB cents per second
}

// We hardcode the pricing list here for now. We shall get them from the portal or AWS API, so that when AWS
// price changes we automatically reflect the new price without a code update. Right now we use the us-east and us-west-2
// pricing, which are the same and which is where we currently deploy. We currently has no indication from platform
// regarding whether this is a on-demand or spot instance. This also has to be enhanced.
var ModelPriceText string = `
t2.micro	1	Variable	1	EBS Only	0.013
t2.small	1	Variable	2	EBS Only	0.026
t2.medium	2	Variable	4	EBS Only	0.052
t2.large	2	Variable	8	EBS Only	0.104
m4.large	2	6.5	8	EBS Only	0.12
m4.xlarge	4	13	16	EBS Only	0.239
m4.2xlarge	8	26	32	EBS Only	0.479
m4.4xlarge	16	53.5	64	EBS Only	0.958
m4.10xlarge	40	124.5	160	EBS Only	2.394
m3.medium	1	3	3.75	1 x 4 SSD	0.067
m3.large	2	6.5	7.5	1 x 32 SSD	0.133
m3.xlarge	4	13	15	2 x 40 SSD	0.266
m3.2xlarge	8	26	30	2 x 80 SSD	0.532
c4.large	2	8	3.75	EBS Only	0.105
c4.xlarge	4	16	7.5	EBS Only	0.209
c4.2xlarge	8	31	15	EBS Only	0.419
c4.4xlarge	16	62	30	EBS Only	0.838
c4.8xlarge	36	132	60	EBS Only	1.675
c3.large	2	7	3.75	2 x 16 SSD	0.105
c3.xlarge	4	14	7.5	2 x 40 SSD	0.21
c3.2xlarge	8	28	15	2 x 80 SSD	0.42
c3.4xlarge	16	55	30	2 x 160 SSD	0.84
c3.8xlarge	32	108	60	2 x 320 SSD	1.68
g2.2xlarge	8	26	15	60 SSD	0.65
g2.8xlarge	32	104	60	2 x 120 SSD	2.6
x1.32xlarge	128	349	1952	2 x 1920 SSD	13.338
r3.large	2	6.5	15	1 x 32 SSD	0.166
r3.xlarge	4	13	30.5	1 x 80 SSD	0.333
r3.2xlarge	8	26	61	1 x 160 SSD	0.665
r3.4xlarge	16	52	122	1 x 320 SSD	1.33
r3.8xlarge	32	104	244	2 x 320 SSD	2.66
i2.xlarge	4	14	30.5	1 x 800 SSD	0.853
i2.2xlarge	8	27	61	2 x 800 SSD	1.705
i2.4xlarge	16	53	122	4 x 800 SSD	3.41
i2.8xlarge	32	104	244	8 x 800 SSD	6.82
d2.xlarge	4	14	30.5	3 x 2000 HDD	0.69
d2.2xlarge	8	28	61	6 x 2000 HDD	1.38
d2.4xlarge	16	56	122	12 x 2000 HDD	2.76
d2.8xlarge	36	116	244	24 x 2000 HDD	5.52
n1-standard-2	2	6.5	7.5	1 x 32 SSD	0.095
`

// Hack for GCP cashboard.

// model name to pricing mapping
var ModelPrice map[string]*EC2Price

func init() {
	_ = "breakpoint"
	m := make(map[string]*EC2Price)
	lineEntries := strings.Split(ModelPriceText, "\n")
	for _, line := range lineEntries {
		arr := strings.Split(line, "\t")
		if len(arr) < 6 {
			continue
		}
		name := arr[0]
		cpu, _ := strconv.Atoi(arr[1])
		mem, _ := strconv.ParseFloat(arr[3], 64)
		cost, _ := strconv.ParseFloat(arr[5], 64)
		cost = cost * 100 / 3600 // convert to cents per second from $ per hour
		m[name] = &EC2Price{
			Cpu:      cpu,
			Mem:      mem,
			Cost:     cost,
			CoreCost: cost / float64(cpu),
			MemCost:  cost / mem / 1024,
		}
	}
	ModelPrice = m
}

var SpendingInterval float64 = 60
