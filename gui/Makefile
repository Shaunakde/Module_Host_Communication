
all:
	build

build:
	docker build -f Dockerfile -t mod_host_gui .

run: build
	docker run -it -p 6379:6379 --network=host mod_host_gui
