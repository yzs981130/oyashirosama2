module openi.cn

go 1.12

require (
	github.com/go-logr/logr v0.1.0
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/robfig/cron v1.2.0
	k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	sigs.k8s.io/controller-runtime v0.2.0
	sigs.k8s.io/controller-tools v0.2.0 // indirect
)

replace k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible => github.com/kubernetes/client-go v11.0.0+incompatible

replace sigs.k8s.io/controller-runtime v0.2.0 => github.com/kubernetes-sigs/controller-runtime v0.2.0

replace golang.org/x/sys v0.0.0-20190429190828-d89cdac9e872 => github.com/golang/sys v0.0.0-20190429190828-d89cdac9e872

replace k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d => github.com/kubernetes/apimachinery v0.0.0-20190404173353-6a84e37a896d

replace golang.org/x/text v0.3.2 => github.com/golang/text v0.3.2

replace golang.org/x/net v0.0.0-20190501004415-9ce7a6920f09 => github.com/golang/net v0.0.0-20190501004415-9ce7a6920f09

replace sigs.k8s.io/controller-tools v0.2.0 => github.com/kubernetes-sigs/controller-tools v0.2.0
