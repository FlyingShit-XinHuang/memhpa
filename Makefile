IMAGE?=flyingshit/mem-hpa

docker-build:
	docker build -t $(IMAGE) .

push: docker-build
	docker push $(IMAGE)

.PHONY: docker-build push
