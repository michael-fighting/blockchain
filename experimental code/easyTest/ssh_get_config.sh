#!/bin/bash

dir=/root/async/
conf="./conf.json"

ips=($(cat ${conf} | jq -r ".branches[].nodes[]" | uniq))

addr=$1

for ((i=0;i<${#ips[@]};i++))
do
  echo "start run ${ips[i]}"
  ssh root@${ips[i]} "cd ${dir} && ./getConfig -addr $addr"
done

echo done