test:
	go test

cover:
	rm -f *.coverprofile
	go test -coverprofile=ratelimiter.coverprofile
	go tool cover -html=ratelimiter.coverprofile

doc:
	godoc -http=:6060

.PHONY: test cover doc
