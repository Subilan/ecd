.PHONY: build run test clean

build:
	cd cli && go build -o ../ecd .

run: build
	./ecd

test:
	cd cli && go vet ./...
	cd cli && go test ./...

clean:
	rm -f ecd
