#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

OW_USER?=guest
OW_RUNTIME?=go:1.15
WSK?=wsk
MAIN=main
SRCS=handler.go go.mod go.sum bencher/function.go
NAME=bencher
ZIP=$(MAIN).zip
MEM?=256
TIMEOUT?=60
TIMEOUT_LIMIT=$(shell echo $(TIMEOUT)\*1000 | bc)

update:
	$(WSK) action update $(NAME) --timeout $(TIMEOUT_LIMIT) --memory $(MEM)

deploy: compile print-TIMEOUT_LIMIT
	$(WSK) action update $(NAME) --timeout $(TIMEOUT_LIMIT) --memory $(MEM) --main $(MAIN) --kind $(OW_RUNTIME) exec.zip

compile: $(SRCS)
	zip exec.zip -qr $(SRCS)

print-%  : ; @echo $($*)

clean:
	-$(WSK) action delete $(NAME)
	-rm exec.zip package.done test.json 2>/dev/null
	-rm test.out 2>/dev/null

info: print-NAME

test: test.json
	$(WSK) action invoke $(NAME) -r | tee -a test.out
	$(WSK) action invoke $(NAME) -P test.json -r | tee -a test.out

test.json:
	echo '{ "name": "Mike" }' >test.json


