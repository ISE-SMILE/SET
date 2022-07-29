.PHONY: build clean deploy copy

# 
MEM_LIMIT=256
TIMEOUT=30
AWS_REGION=eu-central-1

# 
GOFILES=function.go
PYFILES=bencher.py,Pipfile


copy:
	cp workloads/go/$(GOFILES) functions/aws/go/bencher
	cp workloads/go/$(GOFILES) functions/gcf/go/bencher
	cp workloads/go/$(GOFILES) functions/azf/go/bencher
	cp workloads/go/$(GOFILES) functions/ow/go/bencher
	
	cp workloads/python/{$(PYFILES)} functions/aws/python
	cp workloads/python/{$(PYFILES)} functions/gcf/python
	cp workloads/python/{$(PYFILES)} functions/azf/python
	cp workloads/python/{$(PYFILES)} functions/ow/python



go: 
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a --installsuffix cgo --ldflags="-w -s -X main.Build=$(git rev-parse --short HEAD)" -o slet

build: copy go	

clean:
	echo "clean"

deploy: clean build
	
