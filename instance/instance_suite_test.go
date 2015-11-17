package instance_test

import (
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "testing"
    "github.com/Sirupsen/logrus"
)

func TestInstance(t *testing.T) {
    RegisterFailHandler(Fail)
    logrus.SetLevel(logrus.PanicLevel)
    RunSpecs(t, "Instance Suite")
}
