#! /bin/bash

ops=(10000000 100000000) #10000 10000 100000 1000000 
itters=( 10000 10000 100000 1000000)
req=(10 20 40 60)
#echo "Ops,Itters,Recurrsion,ELat,Mem,PageSize,PageFaults,MinorFaults,Swaps" > profile.csv

for o in "${ops[@]}"
do
    for i in "${itters[@]}" 
    do
        for r in "${req[@]}" 
        do
        echo -n "$o,$i,$r," >> profile.csv
        /usr/bin/time -f "%e,%M,%Z,%F,%R,%W" -a -o profile.csv pipenv run python ./test_bencher.py $o $i $r
        done
    done
done
