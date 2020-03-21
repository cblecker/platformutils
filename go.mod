module github.com/cblecker/platformutils

go 1.13

// github.com/openshift/api@release-4.3
require github.com/openshift/api v0.0.0-20200205145930-e9d93e317dd1

require (
	k8s.io/api v0.16.8
	k8s.io/apimachinery v0.16.8
	sigs.k8s.io/controller-runtime v0.4.0
	sigs.k8s.io/yaml v1.2.0
)
