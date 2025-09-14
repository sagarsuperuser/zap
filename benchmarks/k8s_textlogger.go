package benchmarks

import (
	"github.com/go-logr/logr"
	"go.uber.org/zap/internal/ztest"
	"go.uber.org/zap/zapcore"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/textlogger"
)

type kmeta struct {
	Name, Namespace string
}

func (k kmeta) GetName() string {
	return k.Name
}

func (k kmeta) GetNamespace() string {
	return k.Namespace
}

var _ klog.KMetadata = kmeta{}

var KObjSlice = klog.KObjSlice([]interface{}{
	&kmeta{Name: "pod-1", Namespace: "kube-system"},
	&kmeta{Name: "pod-2", Namespace: "kube-system"},
})

func newk8sTextLogger(lvl zapcore.Level) *logr.Logger {
	cfg := textlogger.NewConfig(
		textlogger.Verbosity(0),
		textlogger.Output(&ztest.Discarder{}),
	)

	// Build the logger
	logger := textlogger.NewLogger(cfg)
	return &logger
}
