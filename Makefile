BINARY=dist/rmfakecloud-multiproxy
WINBINARY=dist/rmfakecloud-multiproxy.exe
LINUXBINARY=dist/rmfakecloud-multiproxy64
INSTALLER=dist/installer.sh
.PHONY: clean .FORCE
all: $(INSTALLER) $(WINBINARY) $(LINUXBINARY)
GOSRCFILES=main.go dns.go conf.go

$(LINUXBINARY): version.go $(GOSRCFILES)
	go build -ldflags="-w -s" -o $@

$(BINARY): version.go $(GOSRCFILES)
	GOARCH=arm GOARM=7 go build -ldflags="-w -s" -o $@

$(WINBINARY): version.go $(GOSRCFILES)
	GOOS=windows go build -ldflags="-w -s" -o $@

version.go: generate/versioninfo.go .FORCE
	go generate

$(INSTALLER): $(BINARY) scripts/installer.sh scripts/rmfakecloud-multiproxy.service scripts/rmfakecloudctl
	cp scripts/installer.sh $@
	tar -czvO -C dist `basename $(BINARY)` -C ../scripts rmfakecloud-multiproxy.service rmfakecloudctl >> $@
	chmod +x $@
clean:
	rm -fr dist
