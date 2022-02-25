GOCMD=go
GOBUILD=$(GOCMD) build -buildmode=plugin -trimpath -tags exclude_graphdriver_btrfs
OUTPUT=../../../resources/actions
INSTALL=/usr/local/sysflow/resources/actions

.PHONY: all
all:
	mkdir -p $(OUTPUT); $(GOBUILD) -o $(OUTPUT)/hash.so .  

.PHONY: install
install: all
	mkdir -p $(INSTALL)
	cp $(OUTPUT)/hash.so $(INSTALL)
    

