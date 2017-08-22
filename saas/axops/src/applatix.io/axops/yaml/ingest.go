package yaml

import (
	"encoding/json"
	"hash/fnv"
	"regexp"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/fixture"
	"applatix.io/axops/label"
	"applatix.io/axops/policy"
	"applatix.io/axops/project"
	"applatix.io/axops/service"
	"applatix.io/axops/utils"
	"applatix.io/common"
	"applatix.io/notification_center"
	"applatix.io/template"
)

// Because we use repo:branch as key, all the events in the same repo/branch are serialized. As a result we don't need
// to worry about race conditions of concurrent updates.
//
// If we crashes before finishing updating all the templates, the event will be processed again, and we will still be made whole.
func HandleYamlUpdateEvent(repo string, branch string, revision string, bodyArray []interface{}) *axerror.AXError {

	utils.DebugLog.Printf("Start handling YAMLs of revision %s from %s %s (%d files)\n", revision, repo, branch, len(bodyArray))

	var axErr *axerror.AXError
	ctx := template.NewTemplateBuildContext()
	ctx.IgnoreErrors = true
	ctx.Strict = false
	ctx.Repo = repo
	ctx.Branch = branch
	ctx.Revision = revision

	for _, bodyStr := range bodyArray {
		axErr := ctx.ParseFile([]byte(bodyStr.(string)), "")
		if axErr != nil {
			// We just log the error and proceed. We will end up with a consistent set of correct templates and ignored
			// the ones with errors, and the ones that need them.
			utils.ErrorLog.Printf("Error adding template %v", axErr)
		}
	}
	utils.DebugLog.Println("Validating templates")
	axErr = ctx.Validate()
	if axErr != nil {
		utils.ErrorLog.Printf("Error validating templates %v", axErr)
	}

	axErr = updateServiceTemplates(ctx)
	if axErr != nil {
		utils.ErrorLog.Printf("Error updating service templates %v", axErr)
	}

	axErr = updateFixtureTemplates(ctx)
	if axErr != nil {
		utils.ErrorLog.Printf("Error updating fixtures %v", axErr)
	}

	axErr = updatePolicies(ctx)
	if axErr != nil {
		utils.ErrorLog.Printf("Error updating policies %v", axErr)
	}

	axErr = updateProjects(ctx)
	if axErr != nil {
		utils.ErrorLog.Printf("Error updating projects %v", axErr)
	}

	utils.DebugLog.Printf("Finish handling YAMLs of revision %s from %s %s ......\n", revision, repo, branch)

	service.UpdateTemplateETag()
	policy.UpdateETag()
	project.UpdateETag()
	fixture.UpdateETag()

	return axErr
}

// updateServiceTemplates will insert or update templates of type container, workflow, deployment
func updateServiceTemplates(ctx *template.TemplateBuildContext) *axerror.AXError {
	validatedTemplates := ctx.GetServiceTemplates()
	utils.DebugLog.Printf("Updating %d validated service templates", len(validatedTemplates))

	query := map[string]interface{}{
		service.TemplateRepo:   ctx.Repo,
		service.TemplateBranch: ctx.Branch,
		axdb.AXDBSelectColumns: []string{service.TemplateId, service.TemplateName, service.TemplateType,
			service.TemplateCost, service.TemplateJobsSuccess, service.TemplateJobsFail},
	}
	oldTempArray, axErr := service.GetTemplates(query)
	if axErr != nil {
		return axErr
	}
	utils.DebugLog.Printf("Found %d existing templates in database (repo: %s, branch: %s)", len(oldTempArray), ctx.Repo, ctx.Branch)

	// First iterate existing templates, and see if we need to delete any because either the template is gone, or template was invalid.
	// If there is an existing one, we update the entry in the database (will have a new revision)
	var toBeDeleted []service.EmbeddedTemplateIf
	updated := make(map[string]bool)
	for _, existing := range oldTempArray {
		result, ok := ctx.Results[existing.GetName()]
		if !ok {
			utils.DebugLog.Printf("Marking %v for deletion: no longer exists in branch\n", existing)
			toBeDeleted = append(toBeDeleted, existing)
			continue
		}
		if result.AXErr != nil {
			utils.DebugLog.Printf("Marking %v for deletion: incoming template had error: %v\n", existing, result.AXErr)
			toBeDeleted = append(toBeDeleted, existing)
			continue
		}
		eTmpl, axErr := service.EmbedServiceTemplate(result.Template, ctx)
		if axErr != nil {
			utils.ErrorLog.Printf("Error generating embedded template %s: %v", result.Template.GetName(), axErr)
			continue
		}
		updated[eTmpl.GetName()] = true
		// preserve previous stats
		stats := existing.GetStats()
		eTmpl.SetStats(stats.Cost, stats.JobsFail, stats.JobsSuccess)
		utils.DebugLog.Printf("Updating existing service template %v\n", eTmpl)
		axErr = UpdateTemplate(eTmpl)
		if axErr != nil {
			utils.ErrorLog.Printf("Failed to update template %v: %v", eTmpl, axErr)
		}
	}

	// Iterate the validated templates, and insert any new ones
	for _, st := range validatedTemplates {
		if updated[st.GetName()] {
			// skip the ones we just updated
			continue
		}
		eTmpl, axErr := service.EmbedServiceTemplate(st, ctx)
		if axErr != nil {
			utils.ErrorLog.Printf("Error generating embedded template %s: %v", st.GetName(), axErr)
			continue
		}
		utils.DebugLog.Printf("Inserting new service template %v\n", eTmpl)
		axErr = InsertTemplate(eTmpl)
		if axErr != nil {
			utils.ErrorLog.Printf("Failed to insert template %v: %v", eTmpl, axErr)
		}
	}

	for _, t := range toBeDeleted {
		_, e := utils.Dbcl.Delete(axdb.AXDBAppAXOPS, service.TemplateTable, []map[string]interface{}{{service.TemplateId: t.GetID()}})
		if e != nil {
			utils.ErrorLog.Printf("Failed to delete template %v: %v", t, e)
			axErr = e
		}
	}
	return nil
}

func updatePolicies(ctx *template.TemplateBuildContext) *axerror.AXError {
	var axErr *axerror.AXError
	policyChanged := false
	oldPolicyArray, e := policy.GetPolicies(map[string]interface{}{
		policy.PolicyRepo:   ctx.Repo,
		policy.PolicyBranch: ctx.Branch,
	})
	if e != nil {
		return e
	}

	newPolicy := func(tmpl *template.PolicyTemplate) *policy.Policy {
		return &policy.Policy{
			PolicyTemplate: tmpl,
			Enabled:        utils.NewFalse(),
		}
	}

	var toBeDeleted []policy.Policy
	updated := make(map[string]bool)
	for _, old := range oldPolicyArray {
		new, ok := ctx.Templates[old.Name]
		if !ok || new.GetType() != template.TemplateTypePolicy {
			toBeDeleted = append(toBeDeleted, old)
			continue
		}
		updated[new.GetName()] = true
		updatedPolicy := newPolicy(new.(*template.PolicyTemplate))
		if old.Enabled != nil {
			updatedPolicy.Enabled = old.Enabled
		}

		// If the old policy is an invalid one, that means it was an enabled policy but for some
		//  reason gets deleted or becomes syntax invalid. The new change will put things back
		//  into order, which means makes the policy valid and enabled.
		if old.Status == policy.InvalidStatus {
			utils.DebugLog.Printf("Invalid policy %v (%v) becomes valid. Turn it back on\n", old, old.Enabled)
			updatedPolicy.Status = ""
			updatedPolicy.Enabled = utils.NewTrue()

			// Send notification to notification center
			detail := map[string]interface{}{}
			detail["Policy Url"] = "https://" + common.GetPublicDNS() + "/app/policies/details/" + updatedPolicy.ID
			detail["Policy Name"] = updatedPolicy.Name
			detail["Policy Associated Template"] = updatedPolicy.Template
			detail["Policy Repo"] = updatedPolicy.Repo
			detail["Policy Branch"] = updatedPolicy.Branch
			notification_center.Producer.SendMessage(notification_center.CodeInvalidPolicyBecomesValid, "", []string{}, detail)
		}

		utils.DebugLog.Printf("Update policy %v %v\n", updatedPolicy, updatedPolicy.Enabled)
		_, e := updatedPolicy.Update()
		if e != nil {
			utils.ErrorLog.Printf("Failed to update policy %v: %v", updatedPolicy, e)
			axErr = e
			continue
		}
		policyChanged = true
	}

	for _, pTmpl := range ctx.GetPolicyTemplates() {
		if updated[pTmpl.Name] {
			continue
		}
		newPolicy := newPolicy(pTmpl)
		utils.DebugLog.Printf("Insert new policy %v\n", newPolicy)
		_, e := newPolicy.Insert()
		if e != nil {
			utils.ErrorLog.Printf("Failed to insert policy %v: %v", newPolicy, e)
			axErr = e
		}
	}

	// When deleting policies through GC, we would like to handle the enabled policy gracefully just like updateYAML
	for _, p := range toBeDeleted {
		utils.DebugLog.Printf("Delete policy %v %v\n", p, p.Enabled)
		if p.Enabled != nil && *(p.Enabled) == true {
			// If a policy is enabled, but from the source code, it gets deleted or syntax becomes
			//  invalid for whatever reason, we cannot just delete it without notifying user. The
			//  following change will make the policy invalid and let the user figure out what to do.
			p.Enabled = utils.NewFalse()
			p.Status = policy.InvalidStatus
			_, e := p.Update()
			if e != nil {
				utils.ErrorLog.Printf("Failed to update enabled invalid policy %v: %v", p, e)
				axErr = e
			}
			// Still need to inform jobscheduler as the policy is no longer valid
			policyChanged = true

			// Send notification to notification center
			utils.DebugLog.Printf("Found invalid policy %v %v %v %v %v. Diable the policy and send notification\n", p.Name, p.ID, p.Repo, p.Branch, p.Enabled)
			detail := map[string]interface{}{}
			detail["Policy Url"] = "https://" + common.GetPublicDNS() + "/app/policies/details/" + p.ID
			detail["Policy Name"] = p.Name
			detail["Policy Associated Template"] = p.Template
			detail["Policy Repo"] = p.Repo
			detail["Policy Branch"] = p.Branch
			notification_center.Producer.SendMessage(notification_center.CodeEnabledPolicyInvalid, "", []string{}, detail)
		} else {
			if p.Status != policy.InvalidStatus {
				e := p.Delete()
				if e != nil {
					utils.ErrorLog.Printf("Failed to delete policy %v: %v", p, e)
					axErr = e
				}
				policyChanged = true
			}
		}
	}

	if policyChanged {
		go NotifyScheduleChange(ctx.Repo, ctx.Branch)
	} else {
		utils.DebugLog.Println("No policy change, skip notifying scheduler")
	}
	return axErr
}

func updateFixtureTemplates(ctx *template.TemplateBuildContext) *axerror.AXError {
	fixtureTemplates := ctx.GetFixtureTemplates()

	oldFixtures, axErr := fixture.GetFixtureTemplates(map[string]interface{}{
		fixture.TemplateRepo:   ctx.Repo,
		fixture.TemplateBranch: ctx.Branch,
		axdb.AXDBSelectColumns: []string{fixture.TemplateID, fixture.TemplateName},
	})
	if axErr != nil {
		return axErr
	}

	var fixtureChanged = func(old *template.FixtureTemplate, new *template.FixtureTemplate) bool {
		var hash = func(s string) int64 {
			h := fnv.New32a()
			h.Write([]byte(s))
			return int64(h.Sum32())
		}

		oAttr, _ := json.Marshal(old.Attributes)
		nAttr, _ := json.Marshal(new.Attributes)

		oAction, _ := json.Marshal(old.Actions)
		nAction, _ := json.Marshal(new.Actions)

		return (hash(string(oAttr)) != hash(string(nAttr))) || (hash(string(oAction)) != hash(string(nAction)))
	}

	updateFixtureManager := false
	var toBeDeleted []template.FixtureTemplate
	updated := make(map[string]bool)

	for _, existing := range oldFixtures {
		result, ok := ctx.Results[existing.GetName()]
		if !ok {
			utils.DebugLog.Printf("Marking %v for deletion: no longer exists in branch\n", existing)
			toBeDeleted = append(toBeDeleted, existing)
			continue
		}
		if result.AXErr != nil {
			utils.DebugLog.Printf("Marking %v for deletion: incoming template had error: %v\n", existing, result.AXErr)
			toBeDeleted = append(toBeDeleted, existing)
			continue
		}
		eTmpl := result.Template.(*template.FixtureTemplate)
		utils.DebugLog.Printf("Updating fixture template: %v", eTmpl)
		axErr = fixture.UpsertFixtureTemplate(eTmpl)
		if axErr != nil {
			utils.ErrorLog.Printf("Failed to update template %v: %v", eTmpl, axErr)
		}
		updated[result.Template.GetName()] = true
		updateFixtureManager = updateFixtureManager || fixtureChanged(eTmpl, &existing)
	}

	// upsert templates
	for _, tmpl := range fixtureTemplates {
		if updated[tmpl.Name] {
			continue
		}
		utils.DebugLog.Printf("Inserting new fixture template: %v", tmpl)
		axErr = fixture.UpsertFixtureTemplate(tmpl)
		if axErr != nil {
			utils.ErrorLog.Printf("Failed to update template %v: %v", tmpl, axErr)
		}
	}

	// now delete the templates
	for _, t := range toBeDeleted {
		utils.DebugLog.Printf("Delete fixture template %v\n", t)
		axErr = fixture.DeleteFixtureTemplateByID(t.ID)
		if axErr != nil {
			utils.ErrorLog.Printf("Failed to delete fixture template %v: %v", t, axErr)
		}
	}

	if updateFixtureManager {
		// one ore more fixtures have changed. notify fixture manager
		utils.InfoLog.Println("Notifying fixture manager of updates to fixtures")
		go notifyFixtureChange()
	} else {
		utils.InfoLog.Println("No updates for fixture manager")
	}

	return axErr
}

func notifyFixtureChange() {
	axErr, _ := utils.FixMgrCl.Post2("fixture/template_updates", nil, nil, nil)
	if axErr != nil {
		utils.InfoLog.Printf("Failed to notify fixture manager of updates due to error:%v\n", axErr)
	}
}

func newProject(tmpl *template.ProjectTemplate, ctx *template.TemplateBuildContext) *project.Project {
	generated := project.Project{
		ProjectTemplate: *tmpl,
	}
	//figure out published boolean flag based on filters
	generated.Published = false
	if tmpl.Publish != nil && len(tmpl.Publish.Branches) > 0 {
		for _, v := range tmpl.Publish.Branches {
			if match, _ := regexp.MatchString(v, ctx.Branch); match {
				generated.Published = true
				break
			}
		}
	}

	// asset details
	if tmpl.Assets != nil {
		assets := &project.Assets{}
		if len(tmpl.Assets.Icon) > 0 {
			assets.Icon = &project.AssetDetail{Path: tmpl.Assets.Icon}
		}
		if len(tmpl.Assets.Detail) > 0 {
			assets.Detail = &project.AssetDetail{Path: tmpl.Assets.Detail}
		}
		if len(tmpl.Assets.PublisherIcon) > 0 {
			assets.PublisherIcon = &project.AssetDetail{Path: tmpl.Assets.PublisherIcon}
		}
		generated.Assets = assets
	}
	return &generated
}

func updateProjects(ctx *template.TemplateBuildContext) *axerror.AXError {
	var axErr *axerror.AXError

	oldProjectArray, e := project.GetProjects(map[string]interface{}{
		project.ProjectRepo:   ctx.Repo,
		project.ProjectBranch: ctx.Branch,
	})
	if e != nil {
		return e
	}

	var toBeDeleted []project.Project
	updated := make(map[string]bool)
	for _, old := range oldProjectArray {
		res, exists := ctx.Results[old.Name]
		if !exists || res.AXErr != nil {
			toBeDeleted = append(toBeDeleted, old)
		}
		tmpl := res.Template.(*template.ProjectTemplate)
		updated[tmpl.Name] = true
		new := newProject(tmpl, ctx)
		new.ID = old.ID
		utils.DebugLog.Printf("Update project %v %v %v %v \n", new.Name, new.ID, new.Repo, new.Branch)
		_, e := new.Update()
		if e != nil {
			utils.ErrorLog.Printf("Failed to update project %v: %v", new, e)
			axErr = e
		}
	}

	// delete before insert to avoid asset deletion
	for _, p := range toBeDeleted {
		utils.DebugLog.Printf("Delete project %v %v %v %v\n", p.Name, p.ID, p.Repo, p.Branch)
		e := p.Delete()
		if e != nil {
			utils.ErrorLog.Printf("Failed to delete project %v: %v", p, e)
			axErr = e
		}
	}

	for _, tmpl := range ctx.GetProjectTemplates() {
		if updated[tmpl.Name] {
			continue
		}
		proj := newProject(tmpl, ctx)
		utils.DebugLog.Printf("Insert new project %v %v %v %v\n", proj.Name, proj.ID, proj.Repo, proj.Branch)
		_, e := proj.Insert()
		if e != nil {
			utils.ErrorLog.Printf("Failed to insert project %v: %v", proj, e)
			axErr = e
		}
	}
	return axErr
}

func NotifyScheduleChange(repo, branch string) {
	result := map[string]interface{}{}
	params := map[string]interface{}{
		"repo":   repo,
		"branch": branch,
	}
	e := utils.SchedulerCl.Get("scheduler/refresh", params, &result)
	if e != nil {
		utils.ErrorLog.Printf("Failed to notify the scheduler the policy change, error %v", e)
		//axErr = e
	}
}

func UpdateTemplate(tmpl service.EmbeddedTemplateIf) *axerror.AXError {
	// Persist the template ID
	tempMap := service.TemplateToMap(tmpl)

	for key, value := range tmpl.GetLabels() {
		lb := label.Label{
			Type:  label.LabelTypeService,
			Key:   key,
			Value: value,
		}

		if _, axErr := lb.Create(); axErr != nil {
			if axErr.Code == axerror.ERR_API_DUP_LABEL.Code {
				continue
			} else {
				return axErr
			}
		}
	}

	_, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, service.TemplateTable, tempMap)
	return axErr
}

func InsertTemplate(tmpl service.EmbeddedTemplateIf) *axerror.AXError {
	// if template.Labels == nil {
	// 	template.Labels = map[string]string{}
	// }

	// if template.Annotations == nil {
	// 	template.Annotations = map[string]string{}
	// }

	// Persist the template ID
	tempMap := service.TemplateToMap(tmpl)

	for key, value := range tmpl.GetLabels() {
		lb := label.Label{
			Type:  label.LabelTypeService,
			Key:   key,
			Value: value,
		}

		if _, axErr := lb.Create(); axErr != nil {
			if axErr.Code == axerror.ERR_API_DUP_LABEL.Code {
				continue
			} else {
				return axErr
			}
		}
	}

	_, axErr := utils.Dbcl.Post(axdb.AXDBAppAXOPS, service.TemplateTable, tempMap)
	return axErr
}
