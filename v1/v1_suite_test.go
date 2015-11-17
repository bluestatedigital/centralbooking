package v1_test

import (
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "testing"
    "github.com/Sirupsen/logrus"
)

func TestV1(t *testing.T) {
    RegisterFailHandler(Fail)
    logrus.SetLevel(logrus.PanicLevel)
    RunSpecs(t, "v1 Suite")
}
