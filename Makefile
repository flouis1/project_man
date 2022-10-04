build_linux:
	env GOOS=linux GOARCH=amd64 go build -o project-man
build:
	go build -o project-man

run:
	./project-man diff create psemea fabienl 