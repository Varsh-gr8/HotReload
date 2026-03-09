.PHONY: build demo test clean

build:
	go build -o ./bin/hotreload .

demo: build
	./bin/hotreload \
		--root ./testserver \
		--build "go build -o ./bin/testserver ./testserver" \
		--exec "./bin/testserver"

test:
	go test ./...

clean:
	rm -rf ./bin
