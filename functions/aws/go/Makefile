.PHONY: build clean deploy

build: 
	env CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/tester 

clean:
	rm -rf ./bin

deploy: clean build
	$(shell sls deploy --verbose)

update: build
	$(shell sls deploy)

undeploy:
	$(shell sls remove)

remove: undeploy clean

info:
	$(shell bash -c "sls info 2>/dev/null | grep POST | awk '{print $3}'")