/*
Copyright 2021 The Dapr Authors
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

package utils

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestToISO8601DateTimeString(t *testing.T) {
	t.Run("succeed to convert time.Time to ISO8601 datetime string", func(t *testing.T) {
		testDateTime, err := time.Parse(time.RFC3339, "2020-01-02T15:04:05.123Z")
		assert.NoError(t, err)
		isoString := ToISO8601DateTimeString(testDateTime)
		assert.Equal(t, "2020-01-02T15:04:05.123Z", isoString)
	})

	t.Run("succeed to parse generated iso8601 string to time.Time using RFC3339 Parser", func(t *testing.T) {
		currentTime := time.Unix(1623306411, 123000)
		assert.Equal(t, 123000, currentTime.UTC().Nanosecond())
		isoString := ToISO8601DateTimeString(currentTime)
		assert.Equal(t, "2021-06-10T06:26:51.000123Z", isoString)
		parsed, err := time.Parse(time.RFC3339, isoString)

		// assert
		assert.NoError(t, err)
		assert.Equal(t, currentTime.UTC().Year(), parsed.Year())
		assert.Equal(t, currentTime.UTC().Month(), parsed.Month())
		assert.Equal(t, currentTime.UTC().Day(), parsed.Day())
		assert.Equal(t, currentTime.UTC().Hour(), parsed.Hour())
		assert.Equal(t, currentTime.UTC().Minute(), parsed.Minute())
		assert.Equal(t, currentTime.UTC().Second(), parsed.Second())
		assert.Equal(t, currentTime.UTC().Nanosecond()/1000, parsed.Nanosecond()/1000)
	})
}

func TestParseEnvString(t *testing.T) {
	testCases := []struct {
		testName  string
		envStr    string
		expEnvLen int
		expEnv    []corev1.EnvVar
	}{
		{
			testName:  "empty environment string.",
			envStr:    "",
			expEnvLen: 0,
			expEnv:    []corev1.EnvVar{},
		},
		{
			testName:  "common valid environment string.",
			envStr:    "ENV1=value1,ENV2=value2, ENV3=value3",
			expEnvLen: 3,
			expEnv: []corev1.EnvVar{
				{
					Name:  "ENV1",
					Value: "value1",
				},
				{
					Name:  "ENV2",
					Value: "value2",
				},
				{
					Name:  "ENV3",
					Value: "value3",
				},
			},
		},
		{
			testName:  "special valid environment string.",
			envStr:    `HTTP_PROXY=http://myproxy.com, NO_PROXY="localhost,127.0.0.1,.amazonaws.com"`,
			expEnvLen: 2,
			expEnv: []corev1.EnvVar{
				{
					Name:  "HTTP_PROXY",
					Value: "http://myproxy.com",
				},
				{
					Name:  "NO_PROXY",
					Value: `"localhost,127.0.0.1,.amazonaws.com"`,
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			envVars := ParseEnvString(tc.envStr)
			fmt.Println(tc.testName) //nolint:forbidigo
			assert.Equal(t, tc.expEnvLen, len(envVars))
			assert.Equal(t, tc.expEnv, envVars)
		})
	}
}

func TestParseVolumeMountsString(t *testing.T) {
	testCases := []struct {
		testName     string
		mountStr     string
		readOnly     bool
		expMountsLen int
		expMounts    []corev1.VolumeMount
	}{
		{
			testName:     "empty volume mount string.",
			mountStr:     "",
			readOnly:     false,
			expMountsLen: 0,
			expMounts:    []corev1.VolumeMount{},
		},
		{
			testName:     "valid volume mount string with readonly false.",
			mountStr:     "my-mount:/tmp/mount1,another-mount:/home/user/mount2, mount3:/root/mount3",
			readOnly:     false,
			expMountsLen: 3,
			expMounts: []corev1.VolumeMount{
				{
					Name:      "my-mount",
					MountPath: "/tmp/mount1",
				},
				{
					Name:      "another-mount",
					MountPath: "/home/user/mount2",
				},
				{
					Name:      "mount3",
					MountPath: "/root/mount3",
				},
			},
		},
		{
			testName:     "valid volume mount string with readonly true.",
			mountStr:     " my-mount:/tmp/mount1,mount2:/root/mount2 ",
			readOnly:     true,
			expMountsLen: 2,
			expMounts: []corev1.VolumeMount{
				{
					Name:      "my-mount",
					MountPath: "/tmp/mount1",
					ReadOnly:  true,
				},
				{
					Name:      "mount2",
					MountPath: "/root/mount2",
					ReadOnly:  true,
				},
			},
		},
		{
			testName:     "volume mount string with invalid mounts",
			mountStr:     "my-mount:/tmp/mount1:rw,mount2:/root/mount2,mount3",
			readOnly:     false,
			expMountsLen: 1,
			expMounts: []corev1.VolumeMount{
				{
					Name:      "mount2",
					MountPath: "/root/mount2",
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			mounts := ParseVolumeMountsString(tc.mountStr, tc.readOnly)
			assert.Equal(t, tc.expMountsLen, len(mounts))
			assert.Equal(t, tc.expMounts, mounts)
		})
	}
}

func TestContains(t *testing.T) {
	type customType struct {
		v1 string
		v2 int
	}

	t.Run("find a item", func(t *testing.T) {
		assert.True(t, Contains([]string{"item-1", "item"}, "item"))
		assert.True(t, Contains([]int{1, 2, 3}, 1))
		assert.True(t, Contains([]customType{{v1: "first", v2: 1}, {v1: "second", v2: 2}}, customType{v1: "second", v2: 2}))
	})

	t.Run("didn't find a item", func(t *testing.T) {
		assert.False(t, Contains([]string{"item-1", "item"}, "not-in-item"))
		assert.False(t, Contains([]string{}, "not-in-item"))
		assert.False(t, Contains(nil, "not-in-item"))
		assert.False(t, Contains([]int{1, 2, 3}, 100))
		assert.False(t, Contains([]int{}, 100))
		assert.False(t, Contains(nil, 100))
		assert.False(t, Contains([]customType{{v1: "first", v2: 1}, {v1: "second", v2: 2}}, customType{v1: "foo", v2: 100}))
		assert.False(t, Contains([]customType{}, customType{v1: "foo", v2: 100}))
		assert.False(t, Contains(nil, customType{v1: "foo", v2: 100}))
	})
}

func TestSetEnvVariables(t *testing.T) {
	t.Run("set environment variables success", func(t *testing.T) {
		err := SetEnvVariables(map[string]string{
			"testKey": "testValue",
		})
		assert.Nil(t, err)
		assert.Equal(t, "testValue", os.Getenv("testKey"))
	})
	t.Run("set environment variables failed", func(t *testing.T) {
		err := SetEnvVariables(map[string]string{
			"": "testValue",
		})
		assert.NotNil(t, err)
		assert.NotEqual(t, "testValue", os.Getenv(""))
	})
}

func TestIsYaml(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{
			input:    "a.yaml",
			expected: true,
		}, {
			input:    "a.yml",
			expected: true,
		}, {
			input:    "a.txt",
			expected: false,
		},
	}
	for _, tc := range testCases {
		assert.Equal(t, IsYaml(tc.input), tc.expected)
	}
}
