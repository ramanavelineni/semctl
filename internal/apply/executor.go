package apply

import (
	"fmt"

	"github.com/ramanavelineni/semctl/internal/style"
	apiclient "github.com/ramanavelineni/semctl/pkg/semapi/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/inventory"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/key_store"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/project"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/repository"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/schedule"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/template"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/variable_group"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
)

// Executor applies planned changes to Semaphore.
type Executor struct {
	client *apiclient.Semapi
	config *ApplyConfig
	recon  *Reconciler
}

// NewExecutor creates a new executor.
func NewExecutor(client *apiclient.Semapi, config *ApplyConfig, recon *Reconciler) *Executor {
	return &Executor{
		client: client,
		config: config,
		recon:  recon,
	}
}

// Execute applies all planned actions and returns the number of errors.
func (e *Executor) Execute(plan *Plan) int {
	errors := 0

	// Create/update order: project → keys → envs → repos → inventories → templates → schedules
	createOrder := []ResourceType{
		ResourceProject,
		ResourceKey,
		ResourceEnvironment,
		ResourceRepository,
		ResourceInventory,
		ResourceTemplate,
		ResourceSchedule,
	}

	for _, rt := range createOrder {
		for _, action := range plan.ActionsByType(rt) {
			if action.Action == ActionCreate || action.Action == ActionUpdate {
				if err := e.executeAction(action); err != nil {
					style.Error(fmt.Sprintf("Failed to %s %s %q: %v", action.Action, action.Type, action.Label, err))
					errors++
				}
			}
		}
	}

	// Delete order (reverse): templates → inventories → repos → envs → keys
	// Project never deleted; schedules never deleted
	deleteOrder := []ResourceType{
		ResourceTemplate,
		ResourceInventory,
		ResourceRepository,
		ResourceEnvironment,
		ResourceKey,
	}

	for _, rt := range deleteOrder {
		for _, action := range plan.ActionsByType(rt) {
			if action.Action == ActionDelete {
				if err := e.executeAction(action); err != nil {
					style.Error(fmt.Sprintf("Failed to delete %s %q: %v", action.Type, action.Label, err))
					errors++
				}
			}
		}
	}

	return errors
}

func (e *Executor) executeAction(action ResourceAction) error {
	switch action.Type {
	case ResourceProject:
		return e.executeProject(action)
	case ResourceKey:
		return e.executeKey(action)
	case ResourceEnvironment:
		return e.executeEnvironment(action)
	case ResourceRepository:
		return e.executeRepository(action)
	case ResourceInventory:
		return e.executeInventory(action)
	case ResourceTemplate:
		return e.executeTemplate(action)
	case ResourceSchedule:
		return e.executeSchedule(action)
	default:
		return fmt.Errorf("unknown resource type: %s", action.Type)
	}
}

func (e *Executor) executeProject(action ResourceAction) error {
	if action.Action != ActionCreate {
		return nil
	}

	req := &models.ProjectRequest{
		Name: e.config.Project,
	}

	params := project.NewPostProjectsParams()
	params.Project = req

	resp, err := e.client.Project.PostProjects(params, nil)
	if err != nil {
		return err
	}

	p := resp.GetPayload()
	e.recon.SetProjectID(p.ID)
	style.Success(fmt.Sprintf("Created project %q (ID: %d)", p.Name, p.ID))
	return nil
}

func (e *Executor) executeKey(action ResourceAction) error {
	entry := e.config.Keys[action.Index]
	pid := e.recon.ProjectID()

	switch action.Action {
	case ActionCreate:
		req := &models.AccessKeyRequest{
			ProjectID: pid,
			Name:      entry.Name,
			Type:      entry.Type,
		}
		e.applyKeySecrets(req, entry)

		params := key_store.NewPostProjectProjectIDKeysParams()
		params.ProjectID = pid
		params.AccessKey = req

		resp, err := e.client.KeyStore.PostProjectProjectIDKeys(params, nil)
		if err != nil {
			return err
		}

		k := resp.GetPayload()
		e.recon.KeyIDByName[k.Name] = k.ID
		style.Success(fmt.Sprintf("Created key %q (ID: %d)", k.Name, k.ID))

	case ActionUpdate:
		req := &models.AccessKeyRequest{
			ID:             action.ExistingID,
			ProjectID:      pid,
			Name:           entry.Name,
			Type:           entry.Type,
			OverrideSecret: true,
		}
		e.applyKeySecrets(req, entry)

		params := key_store.NewPutProjectProjectIDKeysKeyIDParams()
		params.ProjectID = pid
		params.KeyID = action.ExistingID
		params.AccessKey = req

		_, err := e.client.KeyStore.PutProjectProjectIDKeysKeyID(params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Updated key %q (ID: %d)", entry.Name, action.ExistingID))

	case ActionDelete:
		params := key_store.NewDeleteProjectProjectIDKeysKeyIDParams()
		params.ProjectID = pid
		params.KeyID = action.ExistingID

		_, err := e.client.KeyStore.DeleteProjectProjectIDKeysKeyID(params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Deleted key %q (ID: %d)", entry.Name, action.ExistingID))
	}
	return nil
}

func (e *Executor) applyKeySecrets(req *models.AccessKeyRequest, entry KeyEntry) {
	switch entry.Type {
	case "ssh":
		if entry.SSH != nil {
			req.SSH = &models.AccessKeyRequestSSH{
				Login:      entry.SSH.Login,
				PrivateKey: entry.SSH.PrivateKey,
				Passphrase: entry.SSH.Passphrase,
			}
		}
	case "login_password":
		if entry.LoginPassword != nil {
			req.LoginPassword = &models.AccessKeyRequestLoginPassword{
				Login:    entry.LoginPassword.Login,
				Password: entry.LoginPassword.Password,
			}
		}
	}
}

func (e *Executor) executeEnvironment(action ResourceAction) error {
	entry := e.config.Environments[action.Index]
	pid := e.recon.ProjectID()

	switch action.Action {
	case ActionCreate:
		req := &models.EnvironmentRequest{
			ProjectID: pid,
			Name:      entry.Name,
			JSON:      entry.JSON,
			Env:       entry.Env,
			Password:  entry.Password,
		}

		params := variable_group.NewPostProjectProjectIDEnvironmentParams()
		params.ProjectID = pid
		params.Environment = req

		resp, err := e.client.VariableGroup.PostProjectProjectIDEnvironment(params, nil)
		if err != nil {
			return err
		}

		env := resp.GetPayload()
		e.recon.EnvIDByName[env.Name] = env.ID
		style.Success(fmt.Sprintf("Created environment %q (ID: %d)", env.Name, env.ID))

	case ActionUpdate:
		req := &models.EnvironmentRequest{
			ID:        action.ExistingID,
			ProjectID: pid,
			Name:      entry.Name,
			JSON:      entry.JSON,
			Env:       entry.Env,
			Password:  entry.Password,
		}

		params := variable_group.NewPutProjectProjectIDEnvironmentEnvironmentIDParams()
		params.ProjectID = pid
		params.EnvironmentID = action.ExistingID
		params.Environment = req

		_, err := e.client.VariableGroup.PutProjectProjectIDEnvironmentEnvironmentID(params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Updated environment %q (ID: %d)", entry.Name, action.ExistingID))

	case ActionDelete:
		params := variable_group.NewDeleteProjectProjectIDEnvironmentEnvironmentIDParams()
		params.ProjectID = pid
		params.EnvironmentID = action.ExistingID

		_, err := e.client.VariableGroup.DeleteProjectProjectIDEnvironmentEnvironmentID(params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Deleted environment %q (ID: %d)", entry.Name, action.ExistingID))
	}
	return nil
}

func (e *Executor) executeRepository(action ResourceAction) error {
	entry := e.config.Repositories[action.Index]
	pid := e.recon.ProjectID()
	sshKeyID := e.recon.resolveKeyID(entry.SSHKey, entry.SSHKeyID)

	switch action.Action {
	case ActionCreate:
		req := &models.RepositoryRequest{
			ProjectID: pid,
			Name:      entry.Name,
			GitURL:    entry.GitURL,
			GitBranch: entry.GitBranch,
			SSHKeyID:  sshKeyID,
		}

		params := repository.NewPostProjectProjectIDRepositoriesParams()
		params.ProjectID = pid
		params.Repository = req

		resp, err := e.client.Repository.PostProjectProjectIDRepositories(params, nil)
		if err != nil {
			return err
		}

		r := resp.GetPayload()
		e.recon.RepoIDByName[r.Name] = r.ID
		style.Success(fmt.Sprintf("Created repository %q (ID: %d)", r.Name, r.ID))

	case ActionUpdate:
		req := &models.RepositoryRequest{
			ID:        action.ExistingID,
			ProjectID: pid,
			Name:      entry.Name,
			GitURL:    entry.GitURL,
			GitBranch: entry.GitBranch,
			SSHKeyID:  sshKeyID,
		}

		params := repository.NewPutProjectProjectIDRepositoriesRepositoryIDParams()
		params.ProjectID = pid
		params.RepositoryID = action.ExistingID
		params.Repository = req

		_, err := e.client.Repository.PutProjectProjectIDRepositoriesRepositoryID(params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Updated repository %q (ID: %d)", entry.Name, action.ExistingID))

	case ActionDelete:
		params := repository.NewDeleteProjectProjectIDRepositoriesRepositoryIDParams()
		params.ProjectID = pid
		params.RepositoryID = action.ExistingID

		_, err := e.client.Repository.DeleteProjectProjectIDRepositoriesRepositoryID(params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Deleted repository %q (ID: %d)", entry.Name, action.ExistingID))
	}
	return nil
}

func (e *Executor) executeInventory(action ResourceAction) error {
	entry := e.config.Inventories[action.Index]
	pid := e.recon.ProjectID()
	sshKeyID := e.recon.resolveKeyID(entry.SSHKey, entry.SSHKeyID)
	becomeKeyID := e.recon.resolveKeyID(entry.BecomeKey, entry.BecomeKeyID)
	repoID := e.recon.resolveRepoID(entry.Repository, entry.RepositoryID)

	switch action.Action {
	case ActionCreate:
		req := &models.InventoryRequest{
			ProjectID:    pid,
			Name:         entry.Name,
			Type:         entry.Type,
			Inventory:    entry.Inventory,
			SSHKeyID:     sshKeyID,
			BecomeKeyID:  becomeKeyID,
			RepositoryID: repoID,
		}

		params := inventory.NewPostProjectProjectIDInventoryParams()
		params.ProjectID = pid
		params.Inventory = req

		resp, err := e.client.Inventory.PostProjectProjectIDInventory(params, nil)
		if err != nil {
			return err
		}

		inv := resp.GetPayload()
		e.recon.InventoryIDByName[inv.Name] = inv.ID
		style.Success(fmt.Sprintf("Created inventory %q (ID: %d)", inv.Name, inv.ID))

	case ActionUpdate:
		req := &models.InventoryRequest{
			ID:           action.ExistingID,
			ProjectID:    pid,
			Name:         entry.Name,
			Type:         entry.Type,
			Inventory:    entry.Inventory,
			SSHKeyID:     sshKeyID,
			BecomeKeyID:  becomeKeyID,
			RepositoryID: repoID,
		}

		params := inventory.NewPutProjectProjectIDInventoryInventoryIDParams()
		params.ProjectID = pid
		params.InventoryID = action.ExistingID
		params.Inventory = req

		_, err := e.client.Inventory.PutProjectProjectIDInventoryInventoryID(params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Updated inventory %q (ID: %d)", entry.Name, action.ExistingID))

	case ActionDelete:
		params := inventory.NewDeleteProjectProjectIDInventoryInventoryIDParams()
		params.ProjectID = pid
		params.InventoryID = action.ExistingID

		_, err := e.client.Inventory.DeleteProjectProjectIDInventoryInventoryID(params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Deleted inventory %q (ID: %d)", entry.Name, action.ExistingID))
	}
	return nil
}

func (e *Executor) executeTemplate(action ResourceAction) error {
	entry := e.config.Templates[action.Index]
	pid := e.recon.ProjectID()
	repoID := e.recon.resolveRepoID(entry.Repository, entry.RepositoryID)
	envID := e.recon.resolveEnvID(entry.Environment, entry.EnvironmentID)
	invID := e.recon.resolveInventoryID(entry.Inventory, entry.InventoryID)
	buildTplID := e.recon.resolveTemplateID(entry.BuildTemplate, entry.BuildTemplateID)

	switch action.Action {
	case ActionCreate:
		req := &models.TemplateRequest{
			ProjectID:               pid,
			Name:                    entry.Name,
			Type:                    entry.Type,
			App:                     entry.App,
			Playbook:                entry.Playbook,
			Description:             entry.Description,
			GitBranch:               entry.GitBranch,
			Arguments:               entry.Arguments,
			StartVersion:            entry.StartVersion,
			Autorun:                 entry.Autorun,
			SuppressSuccessAlerts:   entry.SuppressSuccessAlerts,
			AllowOverrideArgsInTask: entry.AllowOverrideArgsInTask,
			RepositoryID:            repoID,
			EnvironmentID:           envID,
			InventoryID:             invID,
			BuildTemplateID:         buildTplID,
			ViewID:                  entry.ViewID,
		}

		params := template.NewPostProjectProjectIDTemplatesParams()
		params.ProjectID = pid
		params.Template = req

		resp, err := e.client.Template.PostProjectProjectIDTemplates(params, nil)
		if err != nil {
			return err
		}

		t := resp.GetPayload()
		e.recon.TemplateIDByName[t.Name] = t.ID
		style.Success(fmt.Sprintf("Created template %q (ID: %d)", t.Name, t.ID))

	case ActionUpdate:
		req := &models.TemplateRequest{
			ID:                      action.ExistingID,
			ProjectID:               pid,
			Name:                    entry.Name,
			Type:                    entry.Type,
			App:                     entry.App,
			Playbook:                entry.Playbook,
			Description:             entry.Description,
			GitBranch:               entry.GitBranch,
			Arguments:               entry.Arguments,
			StartVersion:            entry.StartVersion,
			Autorun:                 entry.Autorun,
			SuppressSuccessAlerts:   entry.SuppressSuccessAlerts,
			AllowOverrideArgsInTask: entry.AllowOverrideArgsInTask,
			RepositoryID:            repoID,
			EnvironmentID:           envID,
			InventoryID:             invID,
			BuildTemplateID:         buildTplID,
			ViewID:                  entry.ViewID,
		}

		params := template.NewPutProjectProjectIDTemplatesTemplateIDParams()
		params.ProjectID = pid
		params.TemplateID = action.ExistingID
		params.Template = req

		_, err := e.client.Template.PutProjectProjectIDTemplatesTemplateID(params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Updated template %q (ID: %d)", entry.Name, action.ExistingID))

	case ActionDelete:
		params := template.NewDeleteProjectProjectIDTemplatesTemplateIDParams()
		params.ProjectID = pid
		params.TemplateID = action.ExistingID

		_, err := e.client.Template.DeleteProjectProjectIDTemplatesTemplateID(params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Deleted template %q (ID: %d)", entry.Name, action.ExistingID))
	}
	return nil
}

func (e *Executor) executeSchedule(action ResourceAction) error {
	if action.Action != ActionCreate {
		return nil
	}

	entry := e.config.Schedules[action.Index]
	pid := e.recon.ProjectID()
	tplID := e.recon.resolveTemplateID(entry.Template, entry.TemplateID)

	active := true
	if entry.Active != nil {
		active = *entry.Active
	}

	req := &models.ScheduleRequest{
		ProjectID:  pid,
		TemplateID: tplID,
		Name:       entry.Name,
		CronFormat: entry.CronFormat,
		Active:     active,
	}

	params := schedule.NewPostProjectProjectIDSchedulesParams()
	params.ProjectID = pid
	params.Schedule = req

	resp, err := e.client.Schedule.PostProjectProjectIDSchedules(params, nil)
	if err != nil {
		return err
	}

	s := resp.GetPayload()
	style.Success(fmt.Sprintf("Created schedule %q (ID: %d)", s.Name, s.ID))
	return nil
}
