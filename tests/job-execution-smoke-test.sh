#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(readlink -f $(dirname $0))
JOBDECK_API_SERVER_URI=localhost:9090

cd $SCRIPT_DIR/../services/job-api/testdata/grpc

grpcurl -plaintext -d @ $JOBDECK_API_SERVER_URI jobdeck.job.v1.JobService/CreateJob < create_job.json
job_id=$(grpcurl -plaintext -d @ $JOBDECK_API_SERVER_URI jobdeck.job.v1.JobService/ListJobs < list_jobs.json | jq -r '.jobs | map(select(.enabled == true)) | first | .id // empty')
if test -z "${job_id}"; then
    echo "there are no job. creating..."
    grpcurl -plaintext -d @ $JOBDECK_API_SERVER_URI jobdeck.job.v1.JobService/CreateJob < create_job.json
    job_id=$(grpcurl -plaintext -d @ $JOBDECK_API_SERVER_URI jobdeck.job.v1.JobService/ListJobs < list_jobs.json | jq -r '.jobs | map(select(.enabled == true)) | first | .id // empty')
    echo "created."
fi

jq -n --arg job_id "${job_id}" '{jobId: $job_id}' | grpcurl -plaintext -d @ $JOBDECK_API_SERVER_URI jobdeck.job.v1.ExecutionService/DoExecution

echo "added job to queue"
