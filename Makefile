GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
DEP=dep
BINARY_DIR=bin
BINARY_NAME=$(BINARY_DIR)/diskotek

.PHONY: all build deps clean

all: clean build

clean:
	$(GOCLEAN) -v

build:
	$(GOBUILD) -o $(BINARY_NAME) -v

deps:
	$(DEP) ensure -v
