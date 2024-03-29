build:
	go build -o simple.so -buildmode=c-shared .

clean:
	docker compose rm -fsv
run: build
	docker compose up && docker compose rm -fsv
