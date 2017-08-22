package label

const (
	LabelTypeUser    = "user"
	LabelTypeService = "service"
	LabelTypePolicy  = "policy"
	LabelTypeProject = "project"
)

var LabelTypeMap = map[string]bool{
	LabelTypeUser:    true,
	LabelTypeService: true,
	LabelTypePolicy:  true,
	LabelTypeProject: true,
}

const (
	UserLabelCommitter      = "committer" //who last applied the patch
	UserLabelAuthor         = "author"    //who originally wrote the patch
	UserLabelSubmitter      = "submitter" //who submitted the task
	UserLabelSCM            = "scm"
	UserLabelFixtureManager = "fixturemanager"
)
