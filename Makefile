build_linux:
	env GOOS=linux GOARCH=amd64 go build -o proj-sync
build:
	go build -o proj-sync

run:
	./proj-sync diff create psemea fabienl 
