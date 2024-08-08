#!/bin/bash

conf="./conf.json"

ips=($(cat ${conf} | jq -r ".branches[].nodes[]" | uniq))

dir=/root/async/

files=(
"./branch"
"./killall.sh"
"./run.sh"
"./getConfig"
)

for ((i=0;i<${#ips[@]};i++))
do
  ssh root@${ips[i]} mkdir ${dir}
  echo "start to copy files to ${ips[i]}"
  for ((j=0;j<${#files[@]};j++))
  do
    scp ${files[j]} root@${ips[i]}:${dir}${files[j]}
  done
done

echo "done"

