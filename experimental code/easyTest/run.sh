#!/bin/bash

binName=$1
bin=./$binName
outputDir=./output

echo $bin

cnt=$(ps -e | grep $binName | wc -l)
if [ $cnt -ne 0 ]; then
    kill -9 $(ps -e|grep ${binName}|grep -v grep|awk '{print $1}')
    sleep 1
fi

cnt=$(ps -e | grep $binName | wc -l)
if [ $cnt -eq 0 ]; then
    echo "清理上一次执行进程成功"
fi

echo "开始创建输出文件夹"
rm -rf $outputDir
if [ ! -d $outputDir ]; then
  mkdir $outputDir
fi
time=`date +"%Y%m%d-%H%M%S"`
mkdir $outputDir/$time
echo "创建输出文件夹成功"

echo "开始启动节点"
portBase=8000
confDir=./config
for dir in `ls $confDir | sort -k 1.5,1.0n` ;do
    port=$((${dir:4} + ${portBase}))
    echo "start port at ${port}"
    $bin 0.0.0.0 $port $confDir/$dir/config.yml > $outputDir/$time/$dir.log 2>&1 &
done
echo "节点启动成功"