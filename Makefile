build:
	gcloud config set project youneedaspreadsheet
	docker build -t gcr.io/youneedaspreadsheet/app .

deploy: build push
	gcloud config set project youneedaspreadsheet
	gcloud beta run deploy youneedaspreadsheet --image gcr.io/youneedaspreadsheet/app:latest


push:
	gcloud config set project youneedaspreadsheet
	docker push gcr.io/youneedaspreadsheet/app

just-deploy:
	gcloud config set project youneedaspreadsheet
	gcloud beta run deploy banksheets --image gcr.io/youneedaspreadsheet/app:latest
