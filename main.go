/*
Copyright © 2020 Syncier GmbH <info@syncier.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"fmt"
	"os"

	"github.com/syncier/helmt/cmd"
)

var version, commit, date string

func main() {
	err := cmd.Execute(versionString())
	if err != nil {
		os.Exit(1)
	}
}

func versionString() string {
	if version != "" || commit != "" {
		return fmt.Sprintf("%s (Commit: %s, Build Date: %s)", version, commit, date)
	}
	return "?.?.?"
}
