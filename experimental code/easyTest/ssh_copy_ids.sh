#!/bin/bash

ips=(
10.186.77.125
10.186.77.136
10.186.77.139
10.186.77.134
10.186.11.183
10.186.11.193
10.186.11.181
10.186.11.184
10.186.11.197
10.186.11.199
10.186.11.210
10.186.11.39
10.186.11.40
10.186.11.43
10.186.11.4
10.186.11.42
10.186.77.15
10.186.11.192
10.186.11.200
10.186.11.41
10.186.77.120
10.186.77.121
)

passwords=(
gMP7Yb1ldq2TZo63
Zxcvbn2023@
Zxcvbn2023@
Zxcvbn2023@
Zxcvbn2023@
Zxcvbn2023@
Zxcvbn2023@
Zxcvbn2023@
Zxcvbn2023@
Zxcvbn2023@
Zxcvbn2023@
Zxcvbn2023@
Zxcvbn2023@
Zxcvbn2023@
Zxcvbn2023@
Zxcvbn2023@
Zxcvbn2023@
Zxcvbn2023@
Zxcvbn2023@
Zxcvbn2023@
zWMNr3Uy06IV1AOD
fIFRUGzCB8wocWrb
)

for ((i=0;i<${#ips[@]};i++))
do
  echo "$i ssh-copy-id ${ips[i]}"
  sshpass -p"${passwords[i]}" ssh-copy-id -o StrictHostKeyChecking=no root@${ips[i]}
done

echo "done"