#!/bin/bash
procname=$1
echo start to kill $procname
kill -9 $(ps -e|grep ${procname}|grep -v grep|awk '{print $1}')

sleep 1

cnt=$(ps -e | grep $procname | wc -l)
if [ $cnt -ne 0 ]; then
    echo "杀进程失败"
else
    echo "杀进程成功"
fi