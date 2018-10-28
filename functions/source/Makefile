build:
	dep ensure
	env GOOS=linux go build -ldflags="-s -w" -o bin/incident incident/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/estimate estimate/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/cmdb cmdb/main.go
	zip -r ../package/cherwell-service bin
