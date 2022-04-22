IMAGE_NAME := diagrams

run: diagram.png

%.png: %.py build
	docker run -v $$(pwd)/:/tmp/ $(IMAGE_NAME) $<

build: Dockerfile
	docker build . -t $(IMAGE_NAME)
