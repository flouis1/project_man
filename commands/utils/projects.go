package commands

import (
	"errors"
	"io/ioutil"
	"net/http"

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

	if len(c.Arguments) != 4 {
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
//	stDetails, err := commands.GetConfig(conf.dstRt, false)

	log.Warn(rtDetails.Url)
	log.Info(genCurl(rtDetails.Url, rtDetails.AccessToken))

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

////////////////// PROJECT FUNCTIONS

func genCurl(url string, token string) error {

	var result string
	log.Warn(url+"access/api/v1/projects")
	
	// prepare HTTP request
	client := &http.Client{}
	req, err := http.NewRequest("GET", url+"access/api/v1/projects", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// run query and parse it
	resp, err := client.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	} else {
		result = string(body)
	}

	log.Info(result)
	return err
}

func listProjects(){

}

func DoProjectDiff(){

}

