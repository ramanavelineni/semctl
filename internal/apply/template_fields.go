package apply

import (
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
)

// PreserveUnmanagedTemplateFields fills the request fields that neither the
// CLI flags nor apply configs manage — survey vars, vaults, multi-variable-
// group assignments, and task params — from the existing template, so an
// update never wipes them (the server replaces these wholesale on PUT).
// With no existing template it ensures non-nil empty slices, which the API
// requires instead of null. This is THE invariant that has caused wipe bugs
// twice; every TemplateRequest builder must go through it.
func PreserveUnmanagedTemplateFields(req *models.TemplateRequest, existing *models.Template) {
	req.SurveyVars = []*models.TemplateSurveyVar{}
	req.Vaults = []*models.TemplateVault{}
	if existing == nil {
		return
	}
	if existing.SurveyVars != nil {
		req.SurveyVars = existing.SurveyVars
	}
	if existing.Vaults != nil {
		req.Vaults = existing.Vaults
	}
	req.EnvironmentIds = existing.EnvironmentIds
	req.TaskParams = existing.TaskParams
}
