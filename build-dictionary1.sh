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

FILE1=$(mktemp)
FILE2=$(mktemp)
trap "rm -f ${FILE1} ${FILE2}" EXIT

curl --silent --output ${FILE1} --location https://raw.githubusercontent.com/dwyl/english-words/master/words_dictionary.json

sed -i -e '/{/d' -e '/}/d' ${FILE1}
sed -i -e 's/: [0-9],/,/' ${FILE1}
sed -i -e 's/: [0-9]/,/' ${FILE1}

(

cat << '__EOF__'
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

var dictionary = []string{
__EOF__

cat ${FILE1}

echo '}'

) > ${FILE2}

cp ${FILE2} words_dictionary1.go
