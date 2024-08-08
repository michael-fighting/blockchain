#!/bin/bash

#杀进程

dir=/root/async/
conf="./conf.json"

ips=($(cat ${conf} | jq -r ".branches[].nodes[]" | uniq))

for ((i=0;i<${#ips[@]};i++))
do
  echo "start kill ${ips[i]}"
  ssh root@${ips[i]} "cd ${dir} && ./killall.sh branch"
done

echo done
