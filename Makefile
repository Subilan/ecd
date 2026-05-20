.PHONY: build run test clean

build:
	cd cli && go build -o ../ecd-go .

run: build
	./ecd-go

test:
	cd cli && go vet ./...
	cd cli && go test ./...

clean:
	rm -f ecd-go
