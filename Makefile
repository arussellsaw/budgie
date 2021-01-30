build:
	docker build -t gcr.io/russellsaw/banksheets --build-arg BUILDKIT_INLINE_CACHE=1 .

deploy: build push
	gcloud beta run deploy banksheets --image gcr.io/russellsaw/banksheets:latest

deploy-worker: build push
	gcloud beta run deploy banksheets-background --image gcr.io/russellsaw/banksheets:latest

push:
	docker push gcr.io/russellsaw/banksheets

just-deploy:
	gcloud beta run deploy banksheets --image gcr.io/russellsaw/banksheets:latest
