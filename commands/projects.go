package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/jfrog/jfrog-cli-core/v2/common/commands"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"

	"github.com/jfrog/jfrog-client-go/utils/log"
)

func GetDiffCommand() components.Command {
	return components.Command{
		Name:        "diff",
		Description: "provides a repository diff between 2 RT instances",
		Arguments:   getDiffArguments(),
		Flags:       getDiffFlags(),
		Action: func(c *components.Context) error {
			return diffCmd(c)
		},
	}
}

func getDiffArguments() []components.Argument {
	return []components.Argument{
		{
			Name:        "diffType",
			Description: "{create,delete}",
		},
		{
			Name:        "srcRT",
			Description: "Artifactory source (CLI profile name)",
		},
		{
			Name:        "dstRT",
			Description: "Artifactory target (CLI profile name)",
		},
	}
}

func getDiffFlags() []components.Flag {
	return []components.Flag{
		components.StringFlag{
			Name:         "dummy",
			Description:  "dummy",
			DefaultValue: "",
		},
	}
}

////////////////// COMMAND TYPES

type diffConfiguration struct {
	diffType string
	srcRt    string
	dstRt    string
}

const (
	automationRepo   string = "jfrog-automation"
	automationProps  string = "automation=repoName;site=SaaS_DR"
	repoNameFileName string = "RepoNameList.yml"
)

////////////////// MAIN

func diffCmd(c *components.Context) error {

	if len(c.Arguments) != 3 {
		return errors.New("Specify the right number of arguments. Please use --help")
	}

	var conf = new(diffConfiguration)

	conf.diffType = c.Arguments[0]
	conf.srcRt = c.Arguments[1]
	conf.dstRt = c.Arguments[2]
	//var data []byte

	err := checkArgs(conf)
	if err != nil {
		return err
	}

	// init RT connection
	rtDetails, err := commands.GetConfig(conf.srcRt, false)
	stDetails, err := commands.GetConfig(conf.dstRt, false)

	if rtDetails.User != "" {
		return errors.New("Please configure you 'server ID' with a token and not 'User & Password'")
	}

	switch conf.diffType {
	case "create":
		getProjects(rtDetails.Url, rtDetails.AccessToken, stDetails.Url, stDetails.AccessToken)
	case "delete":
		findDeletedProjects(rtDetails.Url, rtDetails.AccessToken, stDetails.Url, stDetails.AccessToken)
	}
	//getProjects(rtDetails.Url, rtDetails.AccessToken, stDetails.Url, stDetails.AccessToken)
	//findDeletedProjects(rtDetails.Url, rtDetails.AccessToken, stDetails.Url, stDetails.AccessToken)
	return nil
}

// Check arguments
func checkArgs(c *diffConfiguration) error {

	switch c.diffType {
	case "create":
	case "delete":
		break
	default:
		return errors.New("Please specify a diffType with these values {create,delete}")
	}

	// check CLI profile existence
	if !validateProfiles(c.srcRt, c.dstRt) {
		return errors.New("One of the profile name couldn't be found")
	}

	return nil
}

// Validates CLI profiles and configured with admin rights
func validateProfiles(src string, dst string) bool {

	found_src := false
	found_dst := false

	for _, profile_name := range commands.GetAllServerIds() {
		if profile_name == src {
			found_src = true
			log.Debug(src + " profile found")
			break
		}
	}

	for _, profile_name := range commands.GetAllServerIds() {
		if profile_name == dst {
			found_dst = true
			log.Debug(dst + " profile found")
			break
		}
	}

	return found_src && found_dst

}

type Project struct {
	DisplayName     string `json:"display_name"`
	Description     string `json:"description"`
	AdminPrivileges struct {
		ManageMembers   bool `json:"manage_members"`
		ManageResources bool `json:"manage_resources"`
		IndexResources  bool `json:"index_resources"`
	} `json:"admin_privileges"`
	StorageQuotaBytes             int    `json:"storage_quota_bytes"`
	SoftLimit                     bool   `json:"soft_limit"`
	StorageQuotaEmailNotification bool   `json:"storage_quota_email_notification"`
	ProjectKey                    string `json:"project_key"`
}

type Users struct {
	Members []struct {
		Name  string   `json:"name"`
		Roles []string `json:"roles"`
	} `json:"members"`
}

type Roles struct {
	Name         string   `json:"name"`
	Actions      []string `json:"actions"`
	Type         string   `json:"type"`
	Environments []string `json:"environments"`
}

func findDeletedProjects(src_url string, src_token string, dst_url string, dst_token string) ([]Project, error) {
	// prepare HTTP request
	client := &http.Client{}
	req, err := http.NewRequest("GET", dst_url+"access/api/v1/projects", nil)
	req.Header.Set("Authorization", "Bearer "+dst_token)

	// run query and parse it
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}
	var project []Project
	json.Unmarshal(body, &project)

	sum := 0
	for i, p := range project {
		getProject(p.ProjectKey, src_url, src_token, dst_url, dst_token)
		sum += i
	}
	return project, err
}

func getProjects(src_url string, src_token string, dst_url string, dst_token string) ([]Project, error) {
	// prepare HTTP request
	client := &http.Client{}
	req, err := http.NewRequest("GET", src_url+"access/api/v1/projects", nil)
	req.Header.Set("Authorization", "Bearer "+src_token)

	// run query and parse it
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}
	var project []Project
	json.Unmarshal(body, &project)

	sum := 0
	for i, p := range project {
		log.Info("------------------------------------------")
		createProject(p.ProjectKey, src_url, src_token, dst_url, dst_token)
		updateProject(p.ProjectKey, src_url, src_token, dst_url, dst_token)
		updateUsers(p.ProjectKey, src_url, src_token, dst_url, dst_token)
		createRoles(p.ProjectKey, src_url, src_token, dst_url, dst_token)
		updateGroups(p.ProjectKey, src_url, src_token, dst_url, dst_token)
		findDeletedRoles(p.ProjectKey, src_url, src_token, dst_url, dst_token)

		sum += i
	}
	return project, err
}

func getProject(project_name string, src_url string, src_token string, dst_url string, dst_token string) error {
	// prepare HTTP request
	client := &http.Client{}
	req, err := http.NewRequest("GET", src_url+"access/api/v1/projects/"+project_name, nil)
	req.Header.Set("Authorization", "Bearer "+src_token)

	// run query and parse it
	resp, err := client.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == 404 {
		log.Info("------------------------------------------")
		log.Info("Project will " + project_name + " be deleted In the following JPD " + dst_url)
		deleteProject(project_name, dst_url, dst_token)
		log.Debug(resp.Status)
		return err
	}else{
		log.Info("Project " + project_name + " exists In the following JPD " + dst_url)
	}
	bytes.NewReader(body)
	return err
}

func createProject(project_name string, src_url string, src_token string, dst_url string, dst_token string) error {
	// prepare HTTP request
	client := &http.Client{}
	req, err := http.NewRequest("GET", src_url+"access/api/v1/projects/"+project_name, nil)
	req.Header.Set("Authorization", "Bearer "+src_token)

	// run query and parse it
	resp, err := client.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	log.Debug("The following configuration will be used : \n" + string(body))
	request_body := bytes.NewReader(body)

	request, err := http.NewRequest("POST", dst_url+"access/api/v1/projects", request_body)
	request.Header.Set("Authorization", "Bearer "+dst_token)
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)

	if err != nil {
		return err
	}
	defer response.Body.Close()

	return err
}

func updateProject(project_name string, src_url string, src_token string, dst_url string, dst_token string) error {
	log.Info("The following project will be updated : " + project_name)
	// prepare HTTP request
	client := &http.Client{}
	req, err := http.NewRequest("GET", src_url+"access/api/v1/projects/"+project_name, nil)
	req.Header.Set("Authorization", "Bearer "+src_token)

	// run query and parse it
	resp, err := client.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	log.Debug("The following configuration will be used : \n" + string(body))
	request_body := bytes.NewReader(body)

	request, err := http.NewRequest("PUT", dst_url+"access/api/v1/projects/"+project_name, request_body)
	request.Header.Set("Authorization", "Bearer "+dst_token)
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)

	if response.StatusCode != 200 {
		log.Error(response)
		return err
	} else {
		log.Info("🐸 Project " + project_name + " is Up to date")
	}
	defer response.Body.Close()

	return err
}

func updateUsers(project_name string, src_url string, src_token string, dst_url string, dst_token string) error {
	// prepare HTTP request
	client := &http.Client{}
	req, err := http.NewRequest("GET", src_url+"access/api/v1/projects/"+project_name+"/users", nil)
	req.Header.Set("Authorization", "Bearer "+src_token)

	// run query and parse it
	resp, err := client.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	var auto Users
	json.Unmarshal(body, &auto)

	sum := 0
	for i, p := range auto.Members {
		createUser(project_name, p.Name, src_url, src_token, dst_url, dst_token)
		sum += i
	}
	return err
}

func createUser(project_name string, user_name string, src_url string, src_token string, dst_url string, dst_token string) error {
	// Get user details in the src JPD
	// prepare HTTP request
	client := &http.Client{}
	req, err := http.NewRequest("GET", src_url+"access/api/v1/projects/"+project_name+"/users/"+user_name, nil)
	req.Header.Set("Authorization", "Bearer "+src_token)

	// run query and parse it
	resp, err := client.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	log.Debug("The following configuration will be used : \n" + string(body))
	request_body := bytes.NewReader(body)

	request, err := http.NewRequest("PUT", dst_url+"access/api/v1/projects/"+project_name+"/users/"+user_name, request_body)
	request.Header.Set("Authorization", "Bearer "+dst_token)
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)

	if response.StatusCode != 200 {
		log.Warn("❌ 👤 " + user_name + " Not Added")
		log.Debug(response)
		return err
	} else {
		log.Info("🐸 👤 " + user_name + " Added")
	}
	defer response.Body.Close()

	return err
}

func updateGroups(project_name string, src_url string, src_token string, dst_url string, dst_token string) error {
	// prepare HTTP request
	client := &http.Client{}
	req, err := http.NewRequest("GET", src_url+"access/api/v1/projects/"+project_name+"/groups", nil)
	req.Header.Set("Authorization", "Bearer "+src_token)

	// run query and parse it
	resp, err := client.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	var auto Users
	json.Unmarshal(body, &auto)

	sum := 0
	for i, p := range auto.Members {
		creategroups(project_name, p.Name, src_url, src_token, dst_url, dst_token)
		sum += i
	}
	return err
}

func creategroups(project_name string, group_name string, src_url string, src_token string, dst_url string, dst_token string) error {
	// Get user details in the src JPD
	// prepare HTTP request
	client := &http.Client{}
	req, err := http.NewRequest("GET", src_url+"access/api/v1/projects/"+project_name+"/groups/"+group_name, nil)
	req.Header.Set("Authorization", "Bearer "+src_token)

	// run query and parse it
	resp, err := client.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	log.Debug("The following configuration will be used : \n" + string(body))
	request_body := bytes.NewReader(body)

	request, err := http.NewRequest("PUT", dst_url+"access/api/v1/projects/"+project_name+"/groups/"+group_name, request_body)
	request.Header.Set("Authorization", "Bearer "+dst_token)
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)

	if response.StatusCode != 200 {
		log.Warn("❌ 👥 " + group_name + " Not Added")
		log.Debug(response)
		return err
	} else {
		log.Info("🐸 👥 " + group_name + " Added")
	}
	defer response.Body.Close()

	return err
}

func listRoles(projectName string, url string, token string) ([]Roles, error) {
    // prepare HTTP request
    client := &http.Client{}
    req, err := http.NewRequest("GET", url+"access/api/v1/projects/"+projectName+"/roles", nil)
    req.Header.Set("Authorization", "Bearer "+token)
    // run query and parse it
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    var auto []Roles
    json.Unmarshal(body, &auto)
    return auto, err
}

func DetailRole(projectName string, url string, token string) ([]Roles, error) {
    // prepare HTTP request
    client := &http.Client{}
    req, err := http.NewRequest("GET", url+"access/api/v1/projects/"+projectName+"/roles", nil)
    req.Header.Set("Authorization", "Bearer "+token)
    // run query and parse it
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    var auto []Roles
    json.Unmarshal(body, &auto)
    return auto, err
}

func createRoles(project_name string, src_url string, src_token string, dst_url string, dst_token string) error {

	roles, err := listRoles(project_name, src_url, src_token)
	sum := 0
	for i, p := range roles {
		manageRoles(project_name, p.Name, src_url, src_token, dst_url, dst_token)
		sum += i
	}
	return err
}

func manageRoles(project_name string, role_name string, src_url string, src_token string, dst_url string, dst_token string) error {
	// prepare HTTP request
	client := http.Client{
		Timeout: 30 * time.Second,
	}
	req, err := http.NewRequest("GET", src_url+"access/api/v1/projects/"+project_name+"/roles/"+role_name, nil)
	req.Header.Set("Authorization", "Bearer "+src_token)

	// run query and parse it
	resp, err := client.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	log.Debug("The following configuration will be used : \n" + string(body))
	request_body := bytes.NewReader(body)

	request, err := http.NewRequest("POST", dst_url+"access/api/v1/projects/"+project_name+"/roles", request_body)
	request.Header.Set("Authorization", "Bearer "+dst_token)
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)

	/* 201: Created
	  409: Conflit (Role mus be updated)
	  400: Bad Request
	  		seen for basics roles (ex: Admin, Release Manager)
	TODO: handle the 400 error code
	*/
	if (response.StatusCode == 201) || (response.StatusCode == 400) {
		log.Info("🐸 Role " + role_name + " Added")
	} else if response.StatusCode == 409 {
		updateRole(project_name, role_name, src_url, src_token, dst_url, dst_token)
	} else {
		log.Error(response.StatusCode)
		log.Warn("❌ Role " + role_name + " Not Added")
		log.Error(response)
		return err
	}
	defer response.Body.Close()

	return err
}

func updateRole(project_name string, role_name string, src_url string, src_token string, dst_url string, dst_token string) error {
	// prepare HTTP request
	client := &http.Client{}
	req, err := http.NewRequest("GET", src_url+"access/api/v1/projects/"+project_name+"/roles/"+role_name, nil)
	req.Header.Set("Authorization", "Bearer "+src_token)

	// run query and parse it
	resp, err := client.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	log.Debug("The following configuration will be used : \n" + string(body))
	request_body := bytes.NewReader(body)

	request, err := http.NewRequest("PUT", dst_url+"access/api/v1/projects/"+project_name+"/roles/"+role_name, request_body)
	request.Header.Set("Authorization", "Bearer "+dst_token)
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if response.StatusCode == 200 {
		log.Info("🐸 Role " + role_name + " Updated")
	} else {
		log.Error(response.StatusCode)
		log.Warn("❌ Role " + role_name + " Not Added")
		log.Debug(response.StatusCode)
		log.Debug(response)
		return err
	}
	defer response.Body.Close()

	return err
}

func findDeletedRoles(project_name string, src_url string, src_token string, dst_url string, dst_token string) error {
	// List roles
	dst_roles, err := listRoles(project_name, dst_url, dst_token)

	// Find roles not in source project
	sum := 0
	for i, s := range dst_roles {
		// prepare HTTP request
		client := http.Client{
			Timeout: 30 * time.Second,
		}
		log.Debug(project_name)
		log.Debug(s.Name)
		req, err := http.NewRequest("GET", src_url+"access/api/v1/projects/"+project_name+"/roles/"+s.Name, nil)
		req.Header.Set("Authorization", "Bearer "+src_token)

		// run query and parse it
		resp, err := client.Do(req)

		if err != nil {
			return err
		}
		defer resp.Body.Close()
		log.Debug(resp.StatusCode)
		sum += i
		if resp.StatusCode == 404{
			log.Info("Role " + s.Name + " Not found in project " + project_name + " of " + src_url + " JPD")
			log.Info("Role " + s.Name + " Will be removed in the ")
			deleteRole(s.Name, project_name, dst_url, dst_token)
		}
	}
	
	// Delete projects
	return err
}

func delProjects(url string, token string,) error {
	// prepare HTTP request
	client := &http.Client{}
	req, err := http.NewRequest("GET", url+"access/api/v1/projects", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// run query and parse it
	resp, err := client.Do(req)

	if err != nil {
		return err
	}
	log.Debug(resp)
	defer resp.Body.Close()

	return err
}

func deleteProject(proj_key string, url string, token string) error {
	// prepare HTTP request
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", url+"access/api/v1/projects/"+proj_key, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// run query and parse it
	resp, err := client.Do(req)

	if err != nil {
		return err
	}
	log.Debug(resp.StatusCode)
	defer resp.Body.Close()
	return err
}

func deleteRole(role_name string, proj_key string, url string, token string) error {
	// prepare HTTP request
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", url + "access/api/v1/projects/"+ proj_key + "/roles/" + role_name, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// run query and parse it
	resp, err := client.Do(req)

	if err != nil {
		return err
	}
	log.Debug(resp.StatusCode)
	defer resp.Body.Close()
	return err
}
