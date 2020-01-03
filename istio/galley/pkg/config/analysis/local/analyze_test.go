// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package local

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"istio.io/istio/galley/pkg/config/analysis"
	"istio.io/istio/galley/pkg/config/analysis/msg"
	"istio.io/istio/galley/pkg/config/meshcfg"
	"istio.io/istio/galley/pkg/config/meta/metadata"
	"istio.io/istio/galley/pkg/config/meta/schema/collection"
	"istio.io/istio/galley/pkg/config/resource"
	"istio.io/istio/galley/pkg/config/source/kube/apiserver"
	"istio.io/istio/galley/pkg/config/source/kube/inmemory"
	"istio.io/istio/galley/pkg/config/testing/basicmeta"
	"istio.io/istio/galley/pkg/config/testing/data"
	"istio.io/istio/galley/pkg/config/testing/k8smeta"
	"istio.io/istio/galley/pkg/config/util/kubeyaml"
	"istio.io/istio/galley/pkg/testing/mock"
)

type testAnalyzer struct {
	fn     func(analysis.Context)
	inputs collection.Names
}

var blankTestAnalyzer = &testAnalyzer{
	fn:     func(_ analysis.Context) {},
	inputs: []collection.Name{},
}

var blankCombinedAnalyzer = analysis.Combine("testCombined", blankTestAnalyzer)

// Metadata implements Analyzer
func (a *testAnalyzer) Metadata() analysis.Metadata {
	return analysis.Metadata{
		Name:   "testAnalyzer",
		Inputs: a.inputs,
	}
}

// Analyze implements Analyzer
func (a *testAnalyzer) Analyze(ctx analysis.Context) {
	a.fn(ctx)
}

func TestAbortWithNoSources(t *testing.T) {
	g := NewGomegaWithT(t)

	cancel := make(chan struct{})

	sa := NewSourceAnalyzer(k8smeta.MustGet(), blankCombinedAnalyzer, "", "", nil, false)
	_, err := sa.Analyze(cancel)
	g.Expect(err).To(Not(BeNil()))
}

func TestAnalyzersRun(t *testing.T) {
	g := NewGomegaWithT(t)

	cancel := make(chan struct{})

	r := createTestResource(t, "ns", "resource", "v1")
	m := msg.NewInternalError(r, "msg")
	a := &testAnalyzer{
		fn: func(ctx analysis.Context) {
			ctx.Exists(data.Collection1, resource.NewFullName("", ""))
			ctx.Report(data.Collection1, m)
		},
	}

	var collectionAccessed collection.Name
	cr := func(col collection.Name) {
		collectionAccessed = col
	}

	sa := NewSourceAnalyzer(metadata.MustGet(), analysis.Combine("a", a), "", "", cr, false)
	err := sa.AddReaderKubeSource(nil)
	g.Expect(err).To(BeNil())

	result, err := sa.Analyze(cancel)
	g.Expect(err).To(BeNil())
	g.Expect(result.Messages).To(ConsistOf(m))
	g.Expect(collectionAccessed).To(Equal(data.Collection1))
	g.Expect(result.ExecutedAnalyzers).To(ConsistOf(a.Metadata().Name))
}

func TestFilterOutputByNamespace(t *testing.T) {
	g := NewGomegaWithT(t)

	cancel := make(chan struct{})

	r1 := createTestResource(t, "ns1", "resource", "v1")
	r2 := createTestResource(t, "ns2", "resource", "v1")
	msg1 := msg.NewInternalError(r1, "msg")
	msg2 := msg.NewInternalError(r2, "msg")
	a := &testAnalyzer{
		fn: func(ctx analysis.Context) {
			ctx.Report(data.Collection1, msg1)
			ctx.Report(data.Collection1, msg2)
		},
	}

	sa := NewSourceAnalyzer(metadata.MustGet(), analysis.Combine("a", a), "ns1", "", nil, false)
	err := sa.AddReaderKubeSource(nil)
	g.Expect(err).To(BeNil())

	result, err := sa.Analyze(cancel)
	g.Expect(err).To(BeNil())
	g.Expect(result.Messages).To(ConsistOf(msg1))
}

func TestAddRunningKubeSource(t *testing.T) {
	g := NewGomegaWithT(t)

	mk := mock.NewKube()

	sa := NewSourceAnalyzer(k8smeta.MustGet(), blankCombinedAnalyzer, "", "", nil, false)

	sa.AddRunningKubeSource(mk)
	g.Expect(sa.sources).To(HaveLen(2))
	g.Expect(sa.sources[0].src).To(BeAssignableToTypeOf(&meshcfg.InMemorySource{})) // Base default meshcfg
	g.Expect(sa.sources[1].src).To(BeAssignableToTypeOf(&apiserver.Source{}))       // All other resources via api server
}

func TestAddRunningKubeSourceWithMeshCfg(t *testing.T) {
	g := NewGomegaWithT(t)

	istioNamespace := resource.Namespace("istio-system")

	cfg := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: meshConfigMapName,
		},
		Data: map[string]string{
			meshConfigMapKey: "",
		},
	}

	mk := mock.NewKube()
	client, err := mk.KubeClient()
	if err != nil {
		t.Fatalf("Error getting client for mock kube: %v", err)
	}
	if _, err := client.CoreV1().ConfigMaps(istioNamespace.String()).Create(cfg); err != nil {
		t.Fatalf("Error creating mesh config configmap: %v", err)
	}

	sa := NewSourceAnalyzer(k8smeta.MustGet(), blankCombinedAnalyzer, "", istioNamespace, nil, false)

	sa.AddRunningKubeSource(mk)
	g.Expect(sa.sources).To(HaveLen(3))
	g.Expect(sa.sources[0].src).To(BeAssignableToTypeOf(&meshcfg.InMemorySource{})) // Base default meshcfg
	g.Expect(sa.sources[1].src).To(BeAssignableToTypeOf(&meshcfg.InMemorySource{})) // in-cluster meshcfg
	g.Expect(sa.sources[2].src).To(BeAssignableToTypeOf(&apiserver.Source{}))       // All other resources via api server
}

func TestAddReaderKubeSource(t *testing.T) {
	g := NewGomegaWithT(t)

	sa := NewSourceAnalyzer(basicmeta.MustGet(), blankCombinedAnalyzer, "", "", nil, false)

	tmpfile := tempFileFromString(t, data.YamlN1I1V1)
	defer os.Remove(tmpfile.Name())

	err := sa.AddReaderKubeSource([]io.Reader{tmpfile})
	g.Expect(err).To(BeNil())
	g.Expect(sa.sources).To(HaveLen(2))
	g.Expect(sa.sources[0].src).To(BeAssignableToTypeOf(&meshcfg.InMemorySource{})) // Base default meshcfg
	g.Expect(sa.sources[1].src).To(BeAssignableToTypeOf(&inmemory.KubeSource{}))    // All other resources via files

	// Note that a blank file for mesh cfg is equivalent to specifying all the defaults
	tmpMeshFile := tempFileFromString(t, "")
	defer func() { _ = os.Remove(tmpMeshFile.Name()) }()

	err = sa.AddFileKubeMeshConfigSource(tmpMeshFile.Name())
	g.Expect(err).To(BeNil())
	g.Expect(sa.sources).To(HaveLen(3))
	g.Expect(sa.sources[2].src).To(BeAssignableToTypeOf(&meshcfg.InMemorySource{})) // meshcfg read from a file
}

func TestAddReaderKubeSourceSkipsBadEntries(t *testing.T) {
	g := NewGomegaWithT(t)

	sa := NewSourceAnalyzer(basicmeta.MustGet(), blankCombinedAnalyzer, "", "", nil, false)

	tmpfile := tempFileFromString(t, kubeyaml.JoinString(data.YamlN1I1V1, "bogus resource entry\n"))
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	err := sa.AddReaderKubeSource([]io.Reader{tmpfile})
	g.Expect(err).To(Not(BeNil()))
	g.Expect(sa.sources).To(HaveLen(2))
}

func TestResourceFiltering(t *testing.T) {
	g := NewGomegaWithT(t)

	// Set up mock apiServer so we can peek at the options it gets started with
	prevApiserverNew := apiserverNew
	defer func() { apiserverNew = prevApiserverNew }()
	var recordedOptions apiserver.Options
	apiserverNew = func(o apiserver.Options) *apiserver.Source {
		recordedOptions = o
		return nil
	}

	usedCollection := k8smeta.K8SCoreV1Services
	a := &testAnalyzer{
		fn:     func(_ analysis.Context) {},
		inputs: []collection.Name{usedCollection},
	}
	mk := mock.NewKube()

	sa := NewSourceAnalyzer(metadata.MustGet(), analysis.Combine("a", a), "", "", nil, true)
	sa.AddRunningKubeSource(mk)

	// All but the used collection should be disabled
	for _, r := range recordedOptions.Resources {
		if r.Collection.Name == usedCollection {
			g.Expect(r.Disabled).To(BeFalse(), fmt.Sprintf("%s should not be disabled", r.Collection.Name))
		} else {
			g.Expect(r.Disabled).To(BeTrue(), fmt.Sprintf("%s should be disabled", r.Collection.Name))
		}
	}
}

func tempFileFromString(t *testing.T, content string) *os.File {
	t.Helper()
	tmpfile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = tmpfile.WriteString(content)
	if err != nil {
		t.Fatal(err)
	}
	err = tmpfile.Sync()
	if err != nil {
		t.Fatal(err)
	}
	_, err = tmpfile.Seek(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	return tmpfile
}
