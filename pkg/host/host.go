// Copyright 2018 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package host

import (
	"github.com/google/syzkaller/prog"
)

// DetectSupportedSyscalls returns list on supported and unsupported syscalls on the host.
// For unsupported syscalls it also returns reason as to why it is unsupported.
func DetectSupportedSyscalls(target *prog.Target, sandbox string) (
	map[*prog.Syscall]bool, map[*prog.Syscall]string, error) {
	supported := make(map[*prog.Syscall]bool)
	unsupported := make(map[*prog.Syscall]string)
	for _, c := range target.Syscalls {
		ok, reason := isSupported(c, sandbox)
		if ok {
			supported[c] = true
		} else {
			if reason == "" {
				reason = "unknown"
			}
			unsupported[c] = reason
		}
	}
	return supported, unsupported, nil
}

const (
	FeatureCoverage = iota
	FeatureComparisons
	FeatureSandboxSetuid
	FeatureSandboxNamespace
	FeatureFaultInjection
	FeatureLeakChecking
	FeatureNetworkInjection
	numFeatures
)

type Feature struct {
	Name    string
	Enabled bool
	Reason  string
}

type Features [numFeatures]Feature

var checkFeature [numFeatures]func() string
var setupFeature [numFeatures]func() error
var callbFeature [numFeatures]func()

func unconditionallyEnabled() string { return "" }

// Check detects features supported on the host.
// Empty string for a feature means the feature is supported,
// otherwise the string contains the reason why the feature is not supported.
func Check() (*Features, error) {
	const unsupported = "support is not implemented in syzkaller"
	res := &Features{
		FeatureCoverage:         {Name: "code coverage", Reason: unsupported},
		FeatureComparisons:      {Name: "comparison tracing", Reason: unsupported},
		FeatureSandboxSetuid:    {Name: "setuid sandbox", Reason: unsupported},
		FeatureSandboxNamespace: {Name: "namespace sandbox", Reason: unsupported},
		FeatureFaultInjection:   {Name: "fault injection", Reason: unsupported},
		FeatureLeakChecking:     {Name: "leak checking", Reason: unsupported},
		FeatureNetworkInjection: {Name: "net packed injection", Reason: unsupported},
	}
	for n, check := range checkFeature {
		if check == nil {
			continue
		}
		if reason := check(); reason == "" {
			res[n].Enabled = true
			res[n].Reason = "enabled"
		} else {
			res[n].Reason = reason
		}
	}
	return res, nil
}

// Setup enables and does any one-time setup for the requested features on the host.
// Note: this can be called multiple times and must be idempotent.
func Setup(features *Features) (func(), error) {
	var callback func()
	for n, setup := range setupFeature {
		if setup == nil || !features[n].Enabled {
			continue
		}
		if err := setup(); err != nil {
			return nil, err
		}
		cb := callbFeature[n]
		if cb != nil {
			prev := callback
			callback = func() {
				cb()
				if prev != nil {
					prev()
				}
			}

		}
	}
	return callback, nil
}
