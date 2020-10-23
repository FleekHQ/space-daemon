package integration_tests_test

import (
	"runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Running App", func() {
	var (
		goRoutinesBeforeAppStart int
	)

	BeforeEach(func() {
		goRoutinesBeforeAppStart = runtime.NumGoroutine()
	})

	It("should start up", func() {
		Expect(2 + 2).To(Equal(4))
	})

	When("App is shutdown", func() {
		It("should not leak goroutines", func() {
			Expect(runtime.NumGoroutine()).To(Equal(goRoutinesBeforeAppStart))
		})
	})

	Describe("Initializing a user", func() {

	})
})
