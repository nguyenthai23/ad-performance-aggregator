.PHONY: build test bench run clean docker

BINARY := aggregator
INPUT  := ad_data.csv
OUTPUT := results

build:
	go build -o $(BINARY) .

test:
	go test -v -race -count=1 ./...

bench:
	go test -bench=. -benchmem -count=3 ./...

run: build
	./$(BINARY) --input $(INPUT) --output $(OUTPUT)

clean:
	rm -f $(BINARY)
	rm -rf $(OUTPUT)

docker:
	docker build -t ad-aggregator .

docker-run: docker
	docker run --rm -v $(PWD)/$(INPUT):/app/$(INPUT) -v $(PWD)/$(OUTPUT):/app/$(OUTPUT) \
		ad-aggregator --input $(INPUT) --output $(OUTPUT)
