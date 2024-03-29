build:
	go build -o simple.so -buildmode=c-shared .

clean:
	docker compose rm -fsv

up:
	docker compose up

run: build up clean
