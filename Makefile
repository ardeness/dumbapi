all:
	export GOPATH=`pwd`
	go get github.com/gorilla/mux
	go get github.com/go-redis/redis
	CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo -o dummapi dummapi.go
