#!/bin/bash

# Pod delete batch size
POD_GC_BATCH=25

# Job delete batch size
JOB_GC_BATCH=10

# Point to your bash-helpers
source /Users/harry/ws1/prod/bash-helpers.sh

# Perform test in default namespace
kns default


create-pod () {
    # create 10 pods
    k create -f testpod.yaml
}

# Targets xlarge standard cluster
#
# WARNING: you might want to run the test first for 2~3min, then the cluster would have ~8K pending Pods.
# Wait for cluster to fully scale up before continue with long running test, as when there are suddenly
# 10s of nodes coming up (which won't be the case as our ADC is controlling pod creaion), master can get
# killed when 8K Pods suddenly get scheduled.
create-job () {
    # create 5 jobs
    k create -f testjob.yaml
}

add-workload () {
    while true; do
        for i in {1..60}; do
            create-pod &
            create-job &
        
            create-pod &
            create-job &
        
            create-pod &
            create-job &

            wait
            sleep 1
        done
        # created 2700 Pods/Jobs
        sleep 180
    done
}

pod-gc () {
    echo "Pod GC started"
    while true; do
        podcnt=0
        pods=""
        pod_gc_pids=""
        for p in `kp -a | grep "Completed\|OutOfpods" | awk '{print $1}'`; do
            podcnt=$(( podcnt + 1 ))
            pods="${pods} ${p}"
            if [[ ${podcnt} -eq ${POD_GC_BATCH} ]]; then
                echo "Pod GC: ${pods}"
                kdp ${pods} &
                pod_gc_pids="${pod_gc_pids} $!"
                pods=""
                podcnt=0
            fi
        done
        if [[ ! -z "${pods}" ]]; then
            echo "Pod GC (Fractions): ${pods}"
            kdp ${pods} &
            pod_gc_pids="${pod_gc_pids} $!"
        fi
        for podgc in ${pod_gc_pids}; do
            echo "Waiting for Pod GC pid ${podgc}"
            wait ${podgc}
        done
        echo "Pod GCed once"
        sleep 5
    done
}

job-gc () {
    echo "Job GC started"
    while true; do
        # Hack: a successful job should have format
        #
        # test-job-pi-zdzp1   1         1            15m
        jobcnt=0
        kjobs=""
        job_gc_pids=""
        for j in `kj -a | grep "1         1" | awk '{print $1}'`; do
            jobcnt=$(( jobcnt + 1 ))
            kjobs="${kjobs} ${j}"
            if [[ ${jobcnt} -eq ${JOB_GC_BATCH} ]]; then
                echo "JOB GC: ${kjobs}"
                k delete jobs ${kjobs} &
                job_gc_pids="${job_jc_pids} $!"
                kjobs=""
                jobcnt=0
            fi
        done

        if [[ ! -z "${kjobs}" ]]; then
            echo "JOB GC (Fractions): ${kjobs}"
            k delete jobs ${kjobs} &
            job_gc_pids="${job_jc_pids} $!"
        fi
        for jobgc in ${job_gc_pids}; do
            echo "Waiting for Job GC pid ${jobgc}"
            wait ${jobgc}
        done
        echo "Job GCed once"
        sleep 5
    done
}

monitor () {
    while true; do
        km dstat -am
    done
}


#add-workload &
monitor &
# add-workload;
pod-gc &
job-gc &

wait

