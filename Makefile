IMAGE = 192.168.1.113/huangxin/tenx-hpa

docker-build:
	docker build -t $(IMAGE) .

push: docker-build
	docker push $(IMAGE)

.PHONY: dev push-dev
