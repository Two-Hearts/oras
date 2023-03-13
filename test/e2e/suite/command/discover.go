/*
Copyright The ORAS Authors.
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

package command

import (
	"encoding/json"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"gopkg.in/yaml.v2"
	"oras.land/oras/test/e2e/internal/testdata/feature"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	"oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
)

func discoverKeyWords(verbose bool, descs ...ocispec.Descriptor) []string {
	var ret []string
	for _, d := range descs {
		ret = append(ret, d.Digest.String(), d.ArtifactType)
		if verbose {
			for k, v := range d.Annotations {
				bytes, err := yaml.Marshal(map[string]string{k: v})
				Expect(err).ShouldNot(HaveOccurred())
				ret = append(ret, strings.TrimSpace(string(bytes)))
			}
		}
	}
	return ret
}

var _ = Describe("ORAS beginners:", func() {
	When("running discover command", func() {
		RunAndShowPreviewInHelp([]string{"discover"})

		It("should show preview and help doc", func() {
			out := ORAS("discover", "--help").MatchKeyWords(feature.Preview.Mark+" Discover", feature.Preview.Description, ExampleDesc).Exec()
			gomega.Expect(out).Should(gbytes.Say("--distribution-spec string\\s+%s", regexp.QuoteMeta(feature.Preview.Mark)))
		})

		It("should fail when no subject reference provided", func() {
			ORAS("discover").ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})

		It("should fail when no tag or digest found in provided subject reference", func() {
			ORAS("discover", RegistryRef(Host, Repo, "")).ExpectFailure().MatchErrKeyWords("Error:", "invalid image reference").Exec()
		})
	})
})

var _ = Describe("Common registry users:", func() {
	subjectRef := RegistryRef(Host, ArtifactRepo, foobar.Tag)
	When("running discover command with json output", func() {
		format := "json"
		It("should discover direct referrers of a subject", func() {
			bytes := ORAS("discover", subjectRef, "-o", format).Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(2))
			Expect(index.Manifests).Should(ContainElement(foobar.SBOMImageReferrer))
			Expect(index.Manifests).Should(ContainElement(foobar.SBOMArtifactReferrer))
		})

		It("should discover matched referrer when filtering", func() {
			bytes := ORAS("discover", subjectRef, "-o", format, "--artifact-type", foobar.SBOMArtifactReferrer.ArtifactType).Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(2))
			Expect(index.Manifests).Should(ContainElement(foobar.SBOMImageReferrer))
			Expect(index.Manifests).Should(ContainElement(foobar.SBOMArtifactReferrer))
		})

		It("should discover matched no referrer", func() {
			bytes := ORAS("discover", subjectRef, "-o", format, "--artifact-type", "???").Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(0))
		})

		It("should discover one referrer with matched platform", func() {
			bytes := ORAS("discover", RegistryRef(Host, ArtifactRepo, multi_arch.Tag), "-o", format, "--platform", "linux/amd64").Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(1))
			Expect(index.Manifests).Should(ContainElement(multi_arch.LinuxAMD64Referrer))
		})
	})

	When("running discover command with tree output", func() {
		format := "tree"
		referrers := []ocispec.Descriptor{foobar.SBOMImageReferrer, foobar.SBOMArtifactReferrer, foobar.SignatureImageReferrer, foobar.SignatureArtifactReferrer}
		It("should discover all referrers of a subject", func() {
			ORAS("discover", subjectRef, "-o", format).
				MatchKeyWords(append(discoverKeyWords(false, referrers...), RegistryRef(Host, ArtifactRepo, foobar.Digest))...).
				Exec()
		})

		It("should discover all referrers of a subject with annotations", func() {
			ORAS("discover", subjectRef, "-o", format, "-v").
				MatchKeyWords(append(discoverKeyWords(true, referrers...), RegistryRef(Host, ArtifactRepo, foobar.Digest))...).
				Exec()

		})
	})
	When("running discover command with table output", func() {
		format := "table"
		It("should all referrers of a subject", func() {
			referrers := []ocispec.Descriptor{foobar.SBOMImageReferrer, foobar.SBOMArtifactReferrer}
			ORAS("discover", subjectRef, "-o", format).
				MatchKeyWords(append(discoverKeyWords(false, referrers...), foobar.Digest)...).
				Exec()
		})
	})
})

var _ = Describe("Fallback registry users:", func() {
	subjectRef := RegistryRef(FallbackHost, ArtifactRepo, foobar.Tag)
	When("running discover command", func() {
		It("should discover direct referrers of a subject via json output", func() {
			bytes := ORAS("discover", subjectRef, "-o", "json").Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(1))
			Expect(index.Manifests).Should(ContainElement(foobar.FallbackSBOMImageReferrer))
		})

		It("should discover matched referrer when filtering via json output", func() {
			bytes := ORAS("discover", subjectRef, "-o", "json", "--artifact-type", foobar.FallbackSBOMImageReferrer.ArtifactType).Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(1))
			Expect(index.Manifests).Should(ContainElement(foobar.FallbackSBOMImageReferrer))
		})

		It("should discover no referrer when not matching via json output", func() {
			bytes := ORAS("discover", subjectRef, "-o", "json", "--artifact-type", "???").Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(0))
		})

		It("should discover all referrers of a subject via tree output", func() {
			referrers := []ocispec.Descriptor{foobar.FallbackSBOMImageReferrer, foobar.FallbackSignatureImageReferrer}
			ORAS("discover", subjectRef, "-o", "tree").
				MatchKeyWords(append(discoverKeyWords(false, referrers...), RegistryRef(FallbackHost, ArtifactRepo, foobar.Digest))...).
				Exec()
		})

		It("should discover all referrers with annotation via tree output", func() {
			referrers := []ocispec.Descriptor{foobar.FallbackSBOMImageReferrer, foobar.FallbackSignatureImageReferrer}
			ORAS("discover", subjectRef, "-o", "tree", "-v").
				MatchKeyWords(append(discoverKeyWords(true, referrers...), RegistryRef(FallbackHost, ArtifactRepo, foobar.Digest))...).
				Exec()
		})
		It("should all referrers of a subject via table output", func() {
			referrers := []ocispec.Descriptor{foobar.FallbackSBOMImageReferrer}
			ORAS("discover", subjectRef, "-o", "table").
				MatchKeyWords(append(discoverKeyWords(false, referrers...), foobar.Digest)...).
				Exec()
		})
	})
})

var _ = Describe("OCI image layout users:", func() {
	prepare := func(subjectRef string) {
		ORAS("cp", RegistryRef(Host, ArtifactRepo, foobar.Tag), subjectRef, Flags.ToLayout, "-r").
			WithDescription("prepare in OCI layout").
			Exec()
	}
	When("running discover command with json output", func() {
		format := "json"
		It("should discover direct referrers of a subject", func() {
			// prepare
			subjectRef := LayoutRef(GinkgoT().TempDir(), foobar.Tag)
			prepare(subjectRef)
			bytes := ORAS("discover", subjectRef, "-o", format, Flags.Layout).Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(2))
			Expect(index.Manifests).Should(ContainElement(foobar.SBOMImageReferrer))
			Expect(index.Manifests).Should(ContainElement(foobar.SBOMArtifactReferrer))
		})

		It("should discover matched referrer when filtering", func() {
			// prepare
			subjectRef := LayoutRef(GinkgoT().TempDir(), foobar.Tag)
			prepare(subjectRef)
			bytes := ORAS("discover", subjectRef, "-o", format, "--artifact-type", foobar.SBOMArtifactReferrer.ArtifactType, Flags.Layout).Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(2))
			Expect(index.Manifests).Should(ContainElement(foobar.SBOMImageReferrer))
			Expect(index.Manifests).Should(ContainElement(foobar.SBOMArtifactReferrer))
		})

		It("should discover no matched referrer", func() {
			// prepare
			subjectRef := LayoutRef(GinkgoT().TempDir(), foobar.Tag)
			prepare(subjectRef)
			bytes := ORAS("discover", subjectRef, "-o", format, "--artifact-type", "???", Flags.Layout).Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(0))
		})
	})

	When("running discover command with tree output", func() {
		format := "tree"
		referrers := []ocispec.Descriptor{foobar.SBOMImageReferrer, foobar.SBOMArtifactReferrer, foobar.SignatureImageReferrer, foobar.SignatureArtifactReferrer}
		It("should discover all referrers of a subject", func() {
			// prepare
			rootDir := GinkgoT().TempDir()
			subjectRef := LayoutRef(rootDir, foobar.Tag)
			prepare(subjectRef)
			ORAS("discover", subjectRef, "-o", format, Flags.Layout).
				MatchKeyWords(append(discoverKeyWords(false, referrers...), LayoutRef(rootDir, foobar.Digest))...).
				Exec()
		})

		It("should discover all referrers of a subject with annotations", func() {
			// prepare
			rootDir := GinkgoT().TempDir()
			subjectRef := LayoutRef(rootDir, foobar.Tag)
			prepare(subjectRef)
			ORAS("discover", subjectRef, "-o", format, "-v", Flags.Layout).
				MatchKeyWords(append(discoverKeyWords(true, referrers...), LayoutRef(rootDir, foobar.Digest))...).
				Exec()
		})
	})

	When("running discover command with table output", func() {
		format := "table"
		It("should direct referrers of a subject", func() {
			referrers := []ocispec.Descriptor{foobar.SBOMImageReferrer, foobar.SBOMArtifactReferrer}
			// prepare
			subjectRef := LayoutRef(GinkgoT().TempDir(), foobar.Tag)
			prepare(subjectRef)
			ORAS("discover", subjectRef, "-o", format, Flags.Layout).
				MatchKeyWords(append(discoverKeyWords(false, referrers...), foobar.Digest)...).
				Exec()
		})

		It("should discover no matched referrer", func() {
			// prepare
			subjectRef := LayoutRef(GinkgoT().TempDir(), foobar.Tag)
			prepare(subjectRef)
			out := ORAS("discover", subjectRef, "-o", format, "--artifact-type", "???", Flags.Layout).Exec().Out
			Expect(out).NotTo(gbytes.Say(foobar.SBOMImageReferrer.Digest.String()))
		})
	})
})
