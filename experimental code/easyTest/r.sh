#!/bin/bash

# 启动进程

dir=/root/async/
conf="./conf.json"

ips=($(cat ${conf} | jq -r ".branches[].nodes[]" | uniq))

for ((i=0;i<${#ips[@]};i++))
do
  echo "start run ${ips[i]}"
  ssh root@${ips[i]} "cd ${dir} && ./run.sh branch"
done

echo done