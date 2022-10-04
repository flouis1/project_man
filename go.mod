module github.com/jfrog/project_man

go 1.14

require (
	// github.com/stretchr/testify v1.8.0
	commands/utils v0.1.0
	github.com/golangci/golangci-lint v1.44.2 // indirect
	github.com/jfrog/jfrog-cli-core/v2 v2.22.1
	github.com/jfrog/jfrog-client-go v1.23.4
	github.com/magiconair/properties v1.8.6
	gopkg.in/yaml.v2 v2.4.0
)

replace commands/utils => ./commands/utils
