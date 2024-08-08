#!/bin/bash
set -e

BLUE='\033[1;34m'
NC='\033[0m'

occlum-go build -buildvcs=false .

server="./server"

if [ ! -f $server ];then
    echo "Error: cannot stat file '$server'"
    echo "Please see README and build it using Occlum Golang toolchain"
    exit 1
fi

# 1. Init Occlum Workspace
rm -rf occlum_instance && mkdir occlum_instance
cd occlum_instance
occlum init
new_json="$(jq '.resource_limits.user_space_size = "2560MB" |
	.resource_limits.kernel_space_heap_size="320MB" |
	.resource_limits.kernel_space_stack_size="10MB" |
	.process.default_stack_size = "40MB" |
	.process.default_heap_size = "320MB" ' Occlum.json)" && \
echo "${new_json}" > Occlum.json

# 2. Copy program into Occlum Workspace and build
rm -rf image
copy_bom -f ../server.yaml --root image --include-dir /opt/occlum/etc/template
occlum build

# 3. Run the web server sample
echo -e "${BLUE}occlum run /bin/server${NC}"
occlum run /bin/server /root/allConfig /root/conf.json
