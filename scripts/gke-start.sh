#!/bin/bash

PROJECT_ID=$(gcloud config get-value project)
CLUSTER="cluster-2"
ZONE="europe-west1-c"

gcloud beta container clusters create $CLUSTER \
       --project $PROJECT_ID \
       --zone $ZONE \
       --machine-type "n1-standard-1" \
       --preemptible
