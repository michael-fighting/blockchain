#!/bin/bash

allConfigDir="./allConfig/"
dir="/root/async/config/"
conf="./conf.json"

ips=($(cat ${conf} | jq -r ".branches[].nodes[]" | uniq))

rm -rf ${dir}
mkdir ${dir}

for ((i=0;i<${#ips[@]};i++))
do
  echo "start clear config dir for ${ips[i]}"
  ssh root@${ips[i]} "rm -rf ${dir}"
  ssh root@${ips[i]} "mkdir ${dir}"
  ssh root@${ips[i]} "systemctl stop firewalld"
done

branchLen=$(cat ${conf} | jq ".branches|length")
index=0

for ((i=0;i<${branchLen};i++))
do
  nodeLen=$(cat ${conf} | jq ".branches["${i}"].nodes|length")
  for ((j=0;j<${nodeLen};j++))
  do
    ip=$(cat ${conf} | jq -r ".branches["${i}"].nodes["${j}"]")
    echo "start to copy node${index} to ${ip}"
    scp -r ${allConfigDir}node${index} root@${ip}:${dir}
    ((index++))
  done
done

echo "done"

