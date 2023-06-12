#!/usr/bin/bash
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

function run_xkcd_passwd() {
#	go run -ldflags="-X main.version=$(git describe --always --long --dirty)" .
	./xkcd-passwd
}

rm --force ./xkcd-passwd
go vet
go build -ldflags="-X main.version=$(git describe --always --long --dirty)" .

(
	while read FILENAME
	do
		echo ${FILENAME}
		cp --force ${FILENAME} defaults.json
		run_xkcd_passwd
	done
) < <(find . -iname 'xkcd-defaults*.json')
