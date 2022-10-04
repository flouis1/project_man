package main

import (
	"github.com/jfrog/jfrog-cli-core/v2/plugins"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/project_man/commands"
)

func main() {
	plugins.PluginMain(getApp())
}

func getApp() components.App {
	app := components.App{}
	app.Name = "project-man"
	app.Description = "Easily greet anyone."
	app.Version = "v0.1.2"
	app.Commands = getCommands()
	return app
}

func getCommands() []components.Command {
	return []components.Command{
		commands.GetDiffCommand(),
		//		commands.GetProvCommand(),
	}
}
