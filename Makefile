BINARY=dist/rmfakecloud-multiproxy
WINBINARY=dist/rmfakecloud-multiproxy.exe
LINUXBINARY=dist/rmfakecloud-multiproxy64
INSTALLER=dist/installer.sh
.PHONY: clean
all: $(INSTALLER) $(WINBINARY) $(LINUXBINARY)

$(LINUXBINARY): version.go main.go
	go build -ldflags="-w -s" -o $@

$(BINARY): version.go main.go
	GOARCH=arm GOARM=7 go build -ldflags="-w -s" -o $@

$(WINBINARY): version.go main.go
	GOOS=windows go build -ldflags="-w -s" -o $@

version.go:
	go generate

$(INSTALLER): $(BINARY) scripts/installer.sh scripts/rmfakecloud-multiproxy.service scripts/rmfakecloudctl
	cp scripts/installer.sh $@
	tar -czvO -C dist `basename $(BINARY)` -C ../scripts rmfakecloud-multiproxy.service rmfakecloudctl >> $@
	chmod +x $@
clean:
	rm -fr dist
