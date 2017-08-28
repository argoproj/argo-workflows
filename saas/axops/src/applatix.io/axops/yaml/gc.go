package yaml

import (
	"applatix.io/axdb"
	"applatix.io/axops/branch"
	"applatix.io/axops/fixture"
	"applatix.io/axops/policy"
	"applatix.io/axops/project"
	"applatix.io/axops/service"
	"applatix.io/axops/utils"
	"applatix.io/common"
	"applatix.io/notification_center"

	"applatix.io/axops/label"
)

const (
	AX_BUILTIN_REPO = "ax-builtin"
)

func GarbageCollectLabels() {

	labels, axErr := label.GetLabels(map[string]interface{}{})
	if axErr != nil {
		utils.ErrorLog.Println("Failed to load all labels for GC:", axErr)
		return
	} else {
		utils.DebugLog.Println("Start processing ", len(labels), " labels")
	}

	for _, lb := range labels {
		utils.DebugLog.Println("Processing label:", lb)

		switch lb.Type {
		case label.LabelTypeUser:
			// Don't GC user label for now
		case label.LabelTypePolicy:
			{
				luceneSearch := axdb.NewLuceneSearch()
				luceneSearch.AddFilterMust(axdb.NewLuceneRegexpFilterBase(policy.PolicyLabels+"$"+lb.Key, lb.Value))
				policies, axErr := policy.GetPolicies(map[string]interface{}{
					axdb.AXDBQuerySearch:     luceneSearch,
					axdb.AXDBQueryMaxEntries: 1,
				})

				if axErr != nil {
					utils.ErrorLog.Println("Failed to query policies for GC:", axErr)
					continue
				}

				if policies == nil || len(policies) == 0 {
					if axErr = lb.Delete(); axErr != nil {
						utils.ErrorLog.Println("Failed to delete label for GC:", axErr)
					} else {
						utils.DebugLog.Println("Label is deleted:", lb)
					}
				}
			}
		case label.LabelTypeProject:
			{
				luceneSearch := axdb.NewLuceneSearch()
				luceneSearch.AddFilterMust(axdb.NewLuceneRegexpFilterBase(project.ProjectLabels+"$"+lb.Key, lb.Value))
				projects, axErr := project.GetProjects(map[string]interface{}{
					axdb.AXDBQuerySearch:     luceneSearch,
					axdb.AXDBQueryMaxEntries: 1,
				})

				if axErr != nil {
					utils.ErrorLog.Println("Failed to query projects for GC:", axErr)
					continue
				}

				if projects == nil || len(projects) == 0 {
					if axErr = lb.Delete(); axErr != nil {
						utils.ErrorLog.Println("Failed to delete label for GC:", axErr)
					} else {
						utils.DebugLog.Println("Label is deleted:", lb)
					}
				}
			}

		case label.LabelTypeService:
			// Check template first
			{
				luceneSearch := axdb.NewLuceneSearch()
				luceneSearch.AddFilterMust(axdb.NewLuceneRegexpFilterBase(service.TemplateLabels+"$"+lb.Key, lb.Value))
				templates, axErr := service.GetTemplates(map[string]interface{}{
					axdb.AXDBQuerySearch:     luceneSearch,
					axdb.AXDBQueryMaxEntries: 1,
				})

				if axErr != nil {
					utils.ErrorLog.Println("Failed to query templates for GC:", axErr)
					continue
				}

				if templates == nil || len(templates) == 0 {

				} else {
					// label is used for template
					continue
				}

			}

			// Check service
			{
				luceneSearch := axdb.NewLuceneSearch()
				luceneSearch.AddFilterMust(axdb.NewLuceneRegexpFilterBase(service.ServiceLabels+"$"+lb.Key, lb.Value))
				services, axErr := service.GetServiceMapsFromDB(map[string]interface{}{
					axdb.AXDBQuerySearch:     luceneSearch,
					axdb.AXDBQueryMaxEntries: 1,
				})

				if axErr != nil {
					utils.ErrorLog.Println("Failed to query services for GC:", axErr)
					continue
				}

				if services == nil || len(services) == 0 {
					if axErr = lb.Delete(); axErr != nil {
						utils.ErrorLog.Println("Failed to delete label for GC:", axErr)
					} else {
						utils.DebugLog.Println("Label is deleted:", lb)
					}
				}
			}
		default:
			utils.ErrorLog.Println("Unexpected label type:", lb.Type)
		}
	}
}

func GarbageCollectTemplatePolicyProjectFixture() {

	branches, axErr := branch.GetBranches(nil)
	if axErr != nil {
		utils.ErrorLog.Println("Failed to load all branches:", axErr)
		return
	}

	utils.DebugLog.Println("GC TEMPLATE POLICY: ", len(branches), " live branches")
	branchesMap := map[string]interface{}{}

	// add ax-builtin repo so that it doesn't get GCed
	branchesMap[AX_BUILTIN_REPO+"_"+AX_BUILTIN_REPO] = "exist"
	for _, b := range branches {
		branchesMap[b.Repo+"_"+b.Name] = "exist"
	}

	templateGC := func(done chan int) {
		defer SendZero(done)
		templates, axErr := service.GetTemplates(
			map[string]interface{}{
				axdb.AXDBSelectColumns: []string{service.TemplateId, service.TemplateBranch, service.TemplateRepo, service.TemplateName},
			},
		)
		if axErr != nil {
			utils.ErrorLog.Println("Failed to load all templates:", axErr)
			return
		}

		utils.DebugLog.Println("GC TEMPLATES: ", len(templates), " existing templates")
		for _, t := range templates {
			if branchesMap[t.GetRepo()+"_"+t.GetBranch()] == nil {
				utils.DebugLog.Printf("GC TEMPLATES: %v %v is gone, deleting the template %v(%v).\n", t.GetRepo(), t.GetBranch(), t.GetName(), t.GetID())
				axErr := service.DeleteTemplateById(t.GetID())
				if axErr != nil {
					utils.DebugLog.Printf("GC TEMPLATES: delete template failed %v.\n", axErr)
				} else {
					utils.DebugLog.Printf("GC TEMPLATES: %v(%v) from %v %v is deleted.\n", t.GetName(), t.GetID(), t.GetRepo(), t.GetBranch())
				}
			}
		}
	}

	policyGC := func(done chan int) {
		defer SendZero(done)
		policies, axErr := policy.GetPolicies(
			map[string]interface{}{
				axdb.AXDBSelectColumns: []string{policy.PolicyID, policy.PolicyName, policy.PolicyRepo, policy.PolicyBranch, policy.PolicyEnabled, policy.PolicyStatus},
			},
		)
		if axErr != nil {
			utils.ErrorLog.Println("Failed to load all policies:", axErr)
			return
		}

		utils.DebugLog.Println("GC POLICIES: ", len(policies), " existing policies")
		for _, p := range policies {
			if branchesMap[p.Repo+"_"+p.Branch] == nil {
				utils.DebugLog.Printf("GC POLICIES: %v %v is gone, deleting the policy %v(%v).\n", p.Repo, p.Branch, p.Name, p.ID)
				if p.Enabled {
					// If a policy is enabled, but from the source code, it gets deleted or syntax becomes
					//  invalid for whatever reason, we cannot just delete it without notifying user. The
					//  following change will make the policy invalid and let the user figure out what to do.
					p.Enabled = false
					p.Status = policy.InvalidStatus
					_, e := p.Update()
					if e != nil {
						utils.ErrorLog.Printf("Failed to update enabled invalid policy %v: %v", p, e)
						axErr = e
					}

					// Send notification to notification center
					utils.DebugLog.Printf("Found invalid policy %v %v %v %v %v during GC. Diable the policy and send notification\n", p.Name, p.ID, p.Repo, p.Branch, p.Enabled)
					detail := map[string]interface{}{}
					detail["Policy Url"] = "https://" + common.GetPublicDNS() + "/app/policies/details/" + p.ID
					detail["Policy Name"] = p.Name
					detail["Policy Associated Template"] = p.Template
					detail["Policy Repo"] = p.Repo
					detail["Policy Branch"] = p.Branch
					detail["Reason"] = "Repo or branch contains this policy gets removed from system."
					notification_center.Producer.SendMessage(notification_center.CodeEnabledPolicyInvalid, "", []string{}, detail)
				} else {
					if p.Status != policy.InvalidStatus {
						e := p.Delete()
						if e != nil {
							utils.ErrorLog.Printf("Failed to delete policy %v: %v", p, e)
							axErr = e
						}
					}
				}
				if axErr != nil {
					utils.DebugLog.Printf("GC POLICIES: delete policy failed %v.\n", axErr)
				} else {
					utils.DebugLog.Printf("GC POLICIES: %v(%v) from %v %v is deleted.\n", p.Name, p.ID, p.Repo, p.Branch)
				}
			}
		}
	}

	projectGC := func(done chan int) {
		defer SendZero(done)
		projects, axErr := project.GetProjects(
			map[string]interface{}{
				axdb.AXDBSelectColumns: []string{project.ProjectID, project.ProjectName, project.ProjectRepo, project.ProjectBranch},
			},
		)
		if axErr != nil {
			utils.ErrorLog.Println("Failed to load all projects:", axErr)
			return
		}

		utils.DebugLog.Println("GC Projects: ", len(projects), " existing projects")
		for _, p := range projects {
			if branchesMap[p.Repo+"_"+p.Branch] == nil {
				utils.DebugLog.Printf("GC Projects: %v %v is gone, deleting the project %v(%v).\n", p.Repo, p.Branch, p.Name, p.ID)
				axErr := p.Delete()
				if axErr != nil {
					utils.DebugLog.Printf("GC Projects: delete project failed %v.\n", axErr)
				} else {
					utils.DebugLog.Printf("GC Projects: %v(%v) from %v %v is deleted.\n", p.Name, p.ID, p.Repo, p.Branch)
				}
			}
		}
	}

	fixtureGC := func(done chan int) {
		defer SendZero(done)
		fixtures, axErr := fixture.GetFixtureTemplates(
			map[string]interface{}{
				axdb.AXDBSelectColumns: []string{fixture.TemplateID, fixture.TemplateName, fixture.TemplateRepo, fixture.TemplateBranch},
			},
		)
		if axErr != nil {
			utils.ErrorLog.Println("Failed to load all fixture templates:", axErr)
			return
		}

		utils.DebugLog.Println("GC fixtures: ", len(fixtures), " existing fixtures")
		for _, f := range fixtures {
			if branchesMap[f.Repo+"_"+f.Branch] == nil {
				utils.DebugLog.Printf("GC fixtures: %v %v is gone, deleting the fixture %v(%v).\n", f.Repo, f.Branch, f.Name, f.ID)
				fixture.DeleteFixtureTemplateByID(f.ID)
				if axErr != nil {
					utils.DebugLog.Printf("GC fixtures: delete fixture failed %v.\n", axErr)
				} else {
					utils.DebugLog.Printf("GC fixtures: %v(%v) from %v %v is deleted.\n", f.Name, f.ID, f.Repo, f.Branch)
				}
			}
		}
	}

	templateDone := make(chan int)
	policyDone := make(chan int)
	projectDone := make(chan int)
	fixtureDone := make(chan int)

	go templateGC(templateDone)
	go policyGC(policyDone)
	go projectGC(projectDone)
	go fixtureGC(fixtureDone)

	_, _, _, _ = <-templateDone, <-policyDone, <-projectDone, fixtureDone

	utils.DebugLog.Println("GC TEMPLATE POLICY PROJECT FIXTURE: done.")
}

func SendZero(done chan int) {
	done <- 0
}
