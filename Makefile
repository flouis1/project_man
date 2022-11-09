build_linux:
	env GOOS=linux GOARCH=amd64 go build -o project_man
build:
	go build -o project_man

run:
	./project_man diff create psemea fabienl 