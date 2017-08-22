package fixture

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
)

func DeleteFixtureCategoryTemplatesByRepo(repo string, best bool) *axerror.AXError {
	utils.InfoLog.Printf("Delete fixture template from %v starting\n", repo)
	fixtures, axErr := GetFixtureTemplates(map[string]interface{}{
		TemplateRepo:           repo,
		axdb.AXDBSelectColumns: []string{TemplateBranch, TemplateName, TemplateRepo, TemplateID},
	})
	utils.InfoLog.Printf("Delete %v fixture templates from %v\n", len(fixtures), repo)

	if axErr != nil {
		if best {
			utils.ErrorLog.Println("Failed to delete branches:", axErr)
		} else {
			return axErr
		}
	}

	for _, fix := range fixtures {
		axErr = DeleteFixtureTemplateByID(fix.ID)
		if axErr != nil {
			if best {
				utils.ErrorLog.Println("Failed to delete branches:", axErr)
			} else {
				return axErr
			}
		}
	}

	utils.InfoLog.Printf("Delete fixture templates from %v finished\n", repo)
	UpdateETag()
	return nil
}
