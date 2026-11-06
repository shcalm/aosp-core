// Copyright 2025 The Android Open Source Project
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"os"

	"google.golang.org/protobuf/proto"
)

func MarshalToFile(from proto.Message, to string) error {
	data, err := proto.Marshal(from)
	if err != nil {
		return err
	}
	if err := os.WriteFile(to, []byte(data), 0644); err != nil {
		return err
	}

	return nil
}

func UnmarshalFile(from string, to proto.Message) error {
	bytes, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	if err := proto.Unmarshal(bytes, to); err != nil {
		return err
	}

	return nil
}
