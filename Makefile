TARGET_CLIENT 	= tcpbc
TARGET_SERVER 	= tcpbs
TARGETS 	  	= ${TARGET_CLIENT} ${TARGET_SERVER}
VERSION 		= `git describe --tags`
BUILD_DATE 		= `date +%F_%T_%z`

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS_CLIENT=-ldflags "-w -s -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE}"
LDFLAGS_SERVER=-ldflags "-w -s -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE}"


.PHONY: default
default: clean ${TARGETS}

${TARGET_CLIENT}:
	go build ${LDFLAGS_CLIENT} -o ${TARGET_CLIENT} -v ./cmd/client

${TARGET_SERVER}: 
	go build ${LDFLAGS_CLIENT} -o ${TARGET_SERVER} -v ./cmd/server

clean: 
	rm -vf $(TARGETS)
	
