PKG = github.com/ginabythebay/poltroon

all: clean data install test

data:
	./install_license_files.sh
	go-bindata -pkg poltroon -prefix data/ data/...

install:
	go install ${PKG}/...

test:
	go test -v ${PKG}/...

clean:
	rm -rf data

.PHONY: all clean data install test
