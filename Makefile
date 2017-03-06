DEV_IMAGE = 192.168.1.113/huangxin/tenx-hpa-dev

dev:
	docker build -f Dockerfile.dev -t $(DEV_IMAGE) .

push-dev: dev
	docker push $(DEV_IMAGE)

.PHONY: dev push-dev
